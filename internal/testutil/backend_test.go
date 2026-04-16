package testutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

func TestMockBackend_Call_ReturnsErr(t *testing.T) {
	mb := &MockBackend{Err: errors.New("boom")}
	err := mb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
	if len(mb.Calls()) != 1 {
		t.Fatalf("Calls length = %d, want 1", len(mb.Calls()))
	}
}

func TestMockBackend_Call_DecodesResponseIntoOut(t *testing.T) {
	mb := &MockBackend{Response: paystack.Response[map[string]string]{
		Status:  true,
		Message: "ok",
		Data:    map[string]string{"reference": "ref_42"},
	}}
	var out paystack.Response[map[string]string]
	if err := mb.Call(context.Background(), http.MethodPost, "/x", nil, &out); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out.Data["reference"] != "ref_42" {
		t.Fatalf("Data = %+v", out.Data)
	}
}

func TestMockBackend_Call_NilOutIsNoop(t *testing.T) {
	mb := &MockBackend{Response: map[string]string{"a": "b"}}
	if err := mb.Call(context.Background(), http.MethodGet, "/x", nil, nil); err != nil {
		t.Fatalf("Call: %v", err)
	}
}

func TestMockBackend_Call_MarshalError(t *testing.T) {
	mb := &MockBackend{Response: make(chan int)}
	var out struct{}
	err := mb.Call(context.Background(), http.MethodGet, "/x", nil, &out)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestMockBackend_LastCall(t *testing.T) {
	mb := &MockBackend{}
	_ = mb.Call(context.Background(), http.MethodGet, "/one", nil, nil)
	_ = mb.Call(context.Background(), http.MethodPost, "/two", "payload", nil)
	last := mb.LastCall()
	if last.Method != http.MethodPost || last.Path != "/two" || last.Params.(string) != "payload" {
		t.Fatalf("LastCall = %+v", last)
	}
}

func TestMockBackend_LastCall_PanicsWhenEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	(&MockBackend{}).LastCall()
}

func TestMockBackend_Reset(t *testing.T) {
	mb := &MockBackend{}
	_ = mb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	mb.Reset()
	if len(mb.Calls()) != 0 {
		t.Fatalf("Calls not reset")
	}
}

func TestMockBackend_CallRaw_DefaultOK(t *testing.T) {
	mb := &MockBackend{}
	resp, err := mb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestMockBackend_CallRaw_UsesConfiguredResponse(t *testing.T) {
	mb := &MockBackend{RawResponse: &http.Response{
		StatusCode: http.StatusCreated,
		Header:     http.Header{"X-Flag": []string{"on"}},
		Body:       http.NoBody,
	}}
	resp, _ := mb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if resp.StatusCode != http.StatusCreated || resp.Header.Get("X-Flag") != "on" {
		t.Fatalf("bad raw response: %+v", resp)
	}
}

func TestMockBackend_CallRaw_ReturnsErr(t *testing.T) {
	mb := &MockBackend{Err: errors.New("nope")}
	_, err := mb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMockBackend_ConcurrentCalls(t *testing.T) {
	mb := &MockBackend{}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
		}()
	}
	wg.Wait()
	if len(mb.Calls()) != 100 {
		t.Fatalf("Calls = %d, want 100", len(mb.Calls()))
	}
}

func TestFixtureBackend_Call_SuccessBody(t *testing.T) {
	fb := &FixtureBackend{Fixture: "error_400_fields.json", Status: http.StatusOK}
	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
	}
	if err := fb.Call(context.Background(), http.MethodGet, "/x", nil, &out); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out.Message != "Validation failed" {
		t.Fatalf("Message = %q", out.Message)
	}
}

func TestFixtureBackend_Call_NilOut(t *testing.T) {
	fb := &FixtureBackend{Fixture: "error_400_fields.json", Status: http.StatusOK}
	if err := fb.Call(context.Background(), http.MethodGet, "/x", nil, nil); err != nil {
		t.Fatalf("Call: %v", err)
	}
}

func TestFixtureBackend_Call_MissingFixtureErrors(t *testing.T) {
	fb := &FixtureBackend{Fixture: "does_not_exist.json", Status: http.StatusOK}
	err := fb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing fixture")
	}
}

func TestFixtureBackend_CallRaw_MissingFixtureErrors(t *testing.T) {
	fb := &FixtureBackend{Fixture: "does_not_exist.json"}
	_, err := fb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err == nil {
		t.Fatal("expected error for missing fixture")
	}
}

func TestFixtureBackend_Call_ErrorStatus(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		status  int
		code    paystack.ErrorCode
		header  http.Header
		retry   time.Duration
	}{
		{"401", "error_401.json", http.StatusUnauthorized, paystack.ErrCodeInvalidKey, nil, 0},
		{"400 fields", "error_400_fields.json", http.StatusBadRequest, paystack.ErrCodeInvalidRequest, nil, 0},
		{"404", "error_404.json", http.StatusNotFound, paystack.ErrCodeNotFound, nil, 0},
		{"429", "error_429.json", http.StatusTooManyRequests, paystack.ErrCodeRateLimited, http.Header{"Retry-After": []string{"5"}}, 5 * time.Second},
		{"500", "error_500.json", http.StatusInternalServerError, paystack.ErrCodeServerError, nil, 0},
		{"HTML 502", "error_html.html", http.StatusBadGateway, paystack.ErrCodeServerError, nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fb := &FixtureBackend{Fixture: tc.fixture, Status: tc.status, Header: tc.header}
			err := fb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
			var pErr *paystack.Error
			if !errors.As(err, &pErr) {
				t.Fatalf("expected *paystack.Error, got %T", err)
			}
			if pErr.Code != tc.code {
				t.Fatalf("Code = %q, want %q", pErr.Code, tc.code)
			}
			if tc.retry != 0 && pErr.RetryAfter != tc.retry {
				t.Fatalf("RetryAfter = %v, want %v", pErr.RetryAfter, tc.retry)
			}
		})
	}
}

func TestFixtureBackend_CallRaw(t *testing.T) {
	fb := &FixtureBackend{Fixture: "error_401.json", Status: http.StatusUnauthorized}
	resp, err := fb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatal("body empty")
	}
}

func TestFixtureBackend_NoFixture(t *testing.T) {
	fb := &FixtureBackend{}
	err := fb.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFixtureBackend_HTMLContentTypeInferred(t *testing.T) {
	fb := &FixtureBackend{Fixture: "error_html.html", Status: http.StatusOK}
	resp, err := fb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}
}

func TestFixtureBackend_ExplicitHeaderWins(t *testing.T) {
	fb := &FixtureBackend{
		Fixture: "error_500.json",
		Status:  http.StatusInternalServerError,
		Header:  http.Header{"Content-Type": []string{"application/problem+json"}},
	}
	resp, _ := fb.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("Content-Type = %q", resp.Header.Get("Content-Type"))
	}
}

func TestLoadFixture(t *testing.T) {
	data := LoadFixture(t, "error_401.json")
	if len(data) == 0 {
		t.Fatal("fixture is empty")
	}
}

func TestLoadFixture_Missing(t *testing.T) {
	sub := &fakeT{}
	defer func() {
		_ = recover()
		if !sub.failed {
			t.Fatal("expected LoadFixture to fail on missing fixture")
		}
	}()
	LoadFixture(sub, "does_not_exist.json")
}

type fakeT struct {
	testing.TB
	failed bool
}

func (f *fakeT) Helper()                                   {}
func (f *fakeT) Fatal(args ...interface{})                 { f.failed = true; panic("fake fatal") }
func (f *fakeT) Fatalf(format string, args ...interface{}) { f.failed = true; panic("fake fatalf") }

func TestRepoRoot_NotFound(t *testing.T) {
	if got := repoRoot("/"); got != "" {
		t.Fatalf("repoRoot(\"/\") = %q, want \"\"", got)
	}
}

func TestRepoRoot_FindsModule(t *testing.T) {
	root := moduleRoot()
	if got := repoRoot(root); got != root {
		t.Fatalf("repoRoot(moduleRoot) = %q, want %q", got, root)
	}
	sub := root + "/internal/testutil"
	if got := repoRoot(sub); got != root {
		t.Fatalf("repoRoot(sub) = %q, want %q", got, root)
	}
}

func TestGuessContentType(t *testing.T) {
	tests := map[string]string{
		"webhook.json":    "application/json",
		"error_html.html": "text/html; charset=utf-8",
		"":                "application/json",
		"weird.txt":       "application/json",
	}
	for name, want := range tests {
		if got := guessContentType(name); got != want {
			t.Fatalf("guessContentType(%q) = %q, want %q", name, got, want)
		}
	}
}
