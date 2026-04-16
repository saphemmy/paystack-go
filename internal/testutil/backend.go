package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	paystack "github.com/saphemmy/paystack-go"
)

// CallRecord captures one invocation of MockBackend.Call or .CallRaw.
type CallRecord struct {
	Method string
	Path   string
	Params interface{}
	Out    interface{}
}

// MockBackend is a Backend double with fully controllable responses. Use it
// when the test is about call shape (method, path, params) rather than
// response parsing.
type MockBackend struct {
	// Response is JSON-encoded into the caller's `out` when non-nil and Err
	// is nil. It can be a ready-made Response[T] / ListResponse[T] envelope
	// or a plain struct — the bytes are marshalled then unmarshalled.
	Response interface{}

	// Err, when set, is returned as-is from Call.
	Err error

	// RawResponse drives CallRaw. When nil, an empty 200 is returned.
	RawResponse *http.Response

	mu    sync.Mutex
	calls []CallRecord
}

var _ paystack.Backend = (*MockBackend)(nil)

// Call records the invocation and either returns Err or JSON-round-trips
// Response into out.
func (m *MockBackend) Call(ctx context.Context, method, path string, params, out interface{}) error {
	m.mu.Lock()
	m.calls = append(m.calls, CallRecord{Method: method, Path: path, Params: params, Out: out})
	m.mu.Unlock()

	if m.Err != nil {
		return m.Err
	}
	if m.Response == nil || out == nil {
		return nil
	}
	buf, err := json.Marshal(m.Response)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, out)
}

// CallRaw records the invocation and returns RawResponse or a default 200.
func (m *MockBackend) CallRaw(ctx context.Context, method, path string, params interface{}) (*http.Response, error) {
	m.mu.Lock()
	m.calls = append(m.calls, CallRecord{Method: method, Path: path, Params: params})
	m.mu.Unlock()

	if m.Err != nil {
		return nil, m.Err
	}
	if m.RawResponse != nil {
		return m.RawResponse, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

// Calls returns a copy of all recorded invocations.
func (m *MockBackend) Calls() []CallRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]CallRecord, len(m.calls))
	copy(out, m.calls)
	return out
}

// LastCall returns the most recent recorded invocation. Panics if none.
func (m *MockBackend) LastCall() CallRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.calls) == 0 {
		panic("testutil: LastCall on empty MockBackend")
	}
	return m.calls[len(m.calls)-1]
}

// Reset clears recorded calls.
func (m *MockBackend) Reset() {
	m.mu.Lock()
	m.calls = nil
	m.mu.Unlock()
}

// FixtureBackend serves a fixture file as if it were a live Paystack
// response. When Status >= 400, the body is passed through paystack.ParseError
// so the returned *Error matches what HTTPBackend would produce.
type FixtureBackend struct {
	// Fixture is a filename under testdata/ (e.g. "transaction_verify.json").
	Fixture string

	// Status is the HTTP status to simulate. Defaults to 200.
	Status int

	// Header lets tests inject response headers (e.g. Retry-After).
	Header http.Header
}

var _ paystack.Backend = (*FixtureBackend)(nil)

// Call loads the fixture, either unmarshals it into out (2xx) or converts
// it to *paystack.Error (4xx/5xx).
func (f *FixtureBackend) Call(ctx context.Context, method, path string, params, out interface{}) error {
	body, resp, err := f.buildResponse()
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return paystack.ParseError(resp, body)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// CallRaw returns a fresh *http.Response built from the fixture.
func (f *FixtureBackend) CallRaw(ctx context.Context, method, path string, params interface{}) (*http.Response, error) {
	body, resp, err := f.buildResponse()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

func (f *FixtureBackend) buildResponse() ([]byte, *http.Response, error) {
	var body []byte
	if f.Fixture != "" {
		loaded, err := readFixture(f.Fixture)
		if err != nil {
			return nil, nil, err
		}
		body = loaded
	}
	resp := &http.Response{
		StatusCode: f.Status,
		Header:     http.Header{},
	}
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}
	for k, vs := range f.Header {
		for _, v := range vs {
			resp.Header.Add(k, v)
		}
	}
	if resp.Header.Get("Content-Type") == "" {
		resp.Header.Set("Content-Type", guessContentType(f.Fixture))
	}
	return body, resp, nil
}

func guessContentType(name string) string {
	if filepath.Ext(name) == ".html" {
		return "text/html; charset=utf-8"
	}
	return "application/json"
}

func readFixture(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(moduleRoot(), "testdata", name))
}
