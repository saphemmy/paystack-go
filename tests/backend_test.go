package paystack_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

type echoRequest struct {
	Method  string
	Path    string
	RawURL  string
	Headers http.Header
	Body    []byte
}

func newEchoServer(t *testing.T, handler func(*echoRequest, http.ResponseWriter)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rec := &echoRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			RawURL:  r.URL.String(),
			Headers: r.Header.Clone(),
			Body:    body,
		}
		handler(rec, w)
	}))
}

type initTxParams struct {
	paystack.Params
	Email  string `json:"email"`
	Amount int64  `json:"amount"`
}

type initTxData struct {
	AuthorizationURL string `json:"authorization_url"`
	Reference        string `json:"reference"`
}

func TestHTTPBackend_Call_SuccessDecodes(t *testing.T) {
	var seen *echoRequest
	srv := newEchoServer(t, func(r *echoRequest, w http.ResponseWriter) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{"authorization_url":"https://checkout.paystack.com/abc","reference":"ref_123"}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	var resp paystack.Response[initTxData]
	err := b.Call(context.Background(), http.MethodPost, "/transaction/initialize", &initTxParams{Email: "c@e.com", Amount: 5000}, &resp)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Data.Reference != "ref_123" {
		t.Fatalf("Reference = %q", resp.Data.Reference)
	}
	if seen.Headers.Get("Authorization") != "Bearer sk_test_xxx" {
		t.Fatalf("Authorization = %q", seen.Headers.Get("Authorization"))
	}
	if seen.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", seen.Headers.Get("Content-Type"))
	}
	if !strings.HasPrefix(seen.Headers.Get("User-Agent"), "paystack-go/") {
		t.Fatalf("User-Agent = %q", seen.Headers.Get("User-Agent"))
	}
	if !strings.Contains(string(seen.Body), `"email":"c@e.com"`) {
		t.Fatalf("body did not carry email: %s", seen.Body)
	}
}

func TestHTTPBackend_Call_ErrorStatuses(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		header     http.Header
		wantCode   paystack.ErrorCode
		wantFields bool
		wantRetry  time.Duration
	}{
		{name: "401", status: 401, body: `{"status":false,"message":"Invalid key"}`, wantCode: paystack.ErrCodeInvalidKey},
		{name: "400 with fields", status: 400, body: `{"status":false,"message":"Validation","errors":{"email":"is required"}}`, wantCode: paystack.ErrCodeInvalidRequest, wantFields: true},
		{name: "404", status: 404, body: `{"status":false,"message":"Not found"}`, wantCode: paystack.ErrCodeNotFound},
		{name: "429", status: 429, body: `{"status":false,"message":"Too many"}`, header: http.Header{"Retry-After": []string{"7"}}, wantCode: paystack.ErrCodeRateLimited, wantRetry: 7 * time.Second},
		{name: "500", status: 500, body: `{"status":false,"message":"Server error"}`, wantCode: paystack.ErrCodeServerError},
		{name: "HTML body", status: 502, body: `<html><body>bad gateway</body></html>`, header: http.Header{"Content-Type": []string{"text/html"}}, wantCode: paystack.ErrCodeServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
				for k, vs := range tc.header {
					for _, v := range vs {
						w.Header().Set(k, v)
					}
				}
				if w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			defer srv.Close()

			b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
			var out paystack.Response[initTxData]
			err := b.Call(context.Background(), http.MethodGet, "/transaction/verify/ref", nil, &out)
			var pErr *paystack.Error
			if !errors.As(err, &pErr) {
				t.Fatalf("expected *Error, got %T: %v", err, err)
			}
			if pErr.Code != tc.wantCode {
				t.Fatalf("Code = %q", pErr.Code)
			}
			if tc.wantFields && len(pErr.Fields) == 0 {
				t.Fatalf("expected Fields to be populated")
			}
			if tc.wantRetry != 0 && pErr.RetryAfter != tc.wantRetry {
				t.Fatalf("RetryAfter = %v, want %v", pErr.RetryAfter, tc.wantRetry)
			}
		})
	}
}

func TestHTTPBackend_Call_IdempotencyKeyHeader(t *testing.T) {
	var seen *echoRequest
	srv := newEchoServer(t, func(r *echoRequest, w http.ResponseWriter) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	key := "idem-abc"
	_ = b.Call(context.Background(), http.MethodPost, "/transaction/initialize", &initTxParams{
		Params: paystack.Params{IdempotencyKey: &key},
		Email:  "x@y.z",
		Amount: 1,
	}, nil)
	if seen.Headers.Get("Idempotency-Key") != key {
		t.Fatalf("Idempotency-Key = %q, want %q", seen.Headers.Get("Idempotency-Key"), key)
	}
	if strings.Contains(string(seen.Body), "IdempotencyKey") || strings.Contains(string(seen.Body), "idempotency_key") {
		t.Fatalf("Idempotency key leaked into body: %s", seen.Body)
	}
}

type listQueryParams struct {
	paystack.ListParams
}

func TestHTTPBackend_Call_ListParamsEncoded(t *testing.T) {
	var seen *echoRequest
	srv := newEchoServer(t, func(r *echoRequest, w http.ResponseWriter) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":[],"meta":{"total":0,"perPage":50,"page":1,"pageCount":0}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	perPage := 25
	page := 3
	_ = b.Call(context.Background(), http.MethodGet, "/transaction", &listQueryParams{
		ListParams: paystack.ListParams{PerPage: &perPage, Page: &page},
	}, nil)

	u, err := url.Parse(seen.RawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if u.Query().Get("perPage") != "25" {
		t.Fatalf("perPage = %q", u.Query().Get("perPage"))
	}
	if u.Query().Get("page") != "3" {
		t.Fatalf("page = %q", u.Query().Get("page"))
	}
}

func TestHTTPBackend_Call_ContextCancelled(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		time.Sleep(2 * time.Second)
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := b.Call(ctx, http.MethodGet, "/slow", nil, nil)
	if err == nil {
		t.Fatal("expected context timeout error")
	}
}

func TestHTTPBackend_Call_DecodeFailureSurfaces(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	var out paystack.Response[initTxData]
	err := b.Call(context.Background(), http.MethodGet, "/x", nil, &out)
	var pErr *paystack.Error
	if !errors.As(err, &pErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if pErr.Code != paystack.ErrCodeServerError {
		t.Fatalf("Code = %q", pErr.Code)
	}
}

func TestHTTPBackend_Call_UnexpectedHTMLOn2xx(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html>unexpected</html>`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	err := b.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	var pErr *paystack.Error
	if !errors.As(err, &pErr) || pErr.Code != paystack.ErrCodeServerError {
		t.Fatalf("expected ErrCodeServerError, got %v", err)
	}
}

func TestHTTPBackend_CallRaw_ReturnsRawResponse(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		w.Header().Set("X-Custom", "yes")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`body`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	resp, err := b.CallRaw(context.Background(), http.MethodGet, "/x", nil)
	if err != nil {
		t.Fatalf("CallRaw: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("StatusCode = %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Custom") != "yes" {
		t.Fatalf("custom header missing")
	}
}

func TestHTTPBackend_LogsViaLeveledLogger(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{}}`))
	})
	defer srv.Close()

	l := &recordingLeveled{}
	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL, LeveledLogger: l})
	_ = b.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	if !strings.Contains(l.debug.String(), "GET") {
		t.Fatalf("expected GET in debug log, got %q", l.debug.String())
	}
}

func TestHTTPBackend_LogsViaPlainLogger(t *testing.T) {
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{}}`))
	})
	defer srv.Close()

	l := &recordingLogger{}
	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL, Logger: l})
	_ = b.Call(context.Background(), http.MethodGet, "/x", nil, nil)
	if !strings.Contains(l.buf.String(), "GET") {
		t.Fatalf("expected GET in log, got %q", l.buf.String())
	}
}

func TestHTTPBackend_ConcurrentCalls(t *testing.T) {
	var hits int64
	srv := newEchoServer(t, func(_ *echoRequest, w http.ResponseWriter) {
		atomic.AddInt64(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Call(context.Background(), http.MethodGet, "/x", nil, nil)
		}()
	}
	wg.Wait()
	if atomic.LoadInt64(&hits) != 50 {
		t.Fatalf("hits = %d, want 50", hits)
	}
}

func TestHTTPBackend_BadMarshalParamsSurfaces(t *testing.T) {
	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: "http://example"})
	type bad struct {
		paystack.Params
		Ch chan int `json:"ch"`
	}
	err := b.Call(context.Background(), http.MethodPost, "/x", &bad{Ch: make(chan int)}, nil)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestHTTPBackend_BadURLBuildSurfaces(t *testing.T) {
	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: "http://example"})
	type badQuery struct {
		Bogus chan int `url:"bogus"`
	}
	err := b.Call(context.Background(), http.MethodGet, "/x", &badQuery{}, nil)
	if err == nil {
		t.Fatal("expected query encode error")
	}
}

func TestHTTPBackend_PathMissingLeadingSlashIsFixed(t *testing.T) {
	var seen *echoRequest
	srv := newEchoServer(t, func(r *echoRequest, w http.ResponseWriter) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	_ = b.Call(context.Background(), http.MethodGet, "transaction", nil, nil)
	if seen.Path != "/transaction" {
		t.Fatalf("Path = %q", seen.Path)
	}
}

func TestHTTPBackend_DeleteEncodesBody(t *testing.T) {
	// DELETE is a write method per the SDK; ensure params marshal into the body
	// when one is provided.
	var seen *echoRequest
	srv := newEchoServer(t, func(r *echoRequest, w http.ResponseWriter) {
		seen = r
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{}}`))
	})
	defer srv.Close()

	b := paystack.NewHTTPBackend("sk_test_xxx", &paystack.BackendConfig{BaseURL: srv.URL})
	_ = b.Call(context.Background(), http.MethodDelete, "/x", &initTxParams{Email: "x@y", Amount: 1}, nil)
	if !strings.Contains(string(seen.Body), "email") {
		t.Fatalf("DELETE body missing email: %s", seen.Body)
	}
}

func TestResponseDataShape(t *testing.T) {
	body := []byte(`{"status":true,"message":"ok","data":{"reference":"r"}}`)
	var r paystack.Response[initTxData]
	if err := json.Unmarshal(body, &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if r.Data.Reference != "r" {
		t.Fatalf("Reference = %q", r.Data.Reference)
	}
}
