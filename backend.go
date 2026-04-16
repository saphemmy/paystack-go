package paystack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	// DefaultBaseURL is the production Paystack API host.
	DefaultBaseURL = "https://api.paystack.co"

	sdkVersion       = "0.1.0"
	userAgentPrefix  = "paystack-go"
	maxResponseBytes = 10 << 20 // 10 MiB cap on any single response body
	defaultTimeout   = 60 * time.Second
)

// Backend is the HTTP contract the SDK depends on. Swap it entirely for
// proxies, recording backends, or test doubles.
type Backend interface {
	Call(ctx context.Context, method, path string, params, out interface{}) error
	CallRaw(ctx context.Context, method, path string, params interface{}) (*http.Response, error)
}

// BackendConfig configures an HTTPBackend. All fields are optional; a nil
// config yields the default production client.
type BackendConfig struct {
	HTTPClient    *http.Client
	BaseURL       string
	Logger        Logger
	LeveledLogger LeveledLogger
}

// HTTPBackend is the default Backend implementation. It is exported so
// integration packages and advanced callers can embed or wrap it.
type HTTPBackend struct {
	key     string
	client  *http.Client
	base    string
	log     Logger
	leveled LeveledLogger
}

// NewHTTPBackend builds an HTTPBackend. cfg may be nil.
func NewHTTPBackend(key string, cfg *BackendConfig) *HTTPBackend {
	b := &HTTPBackend{key: key, base: DefaultBaseURL}
	if cfg != nil {
		if cfg.HTTPClient != nil {
			b.client = cfg.HTTPClient
		}
		if cfg.BaseURL != "" {
			b.base = strings.TrimRight(cfg.BaseURL, "/")
		}
		b.log = cfg.Logger
		b.leveled = cfg.LeveledLogger
	}
	if b.client == nil {
		b.client = &http.Client{Timeout: defaultTimeout}
	}
	return b
}

// Call issues an HTTP request, decodes the JSON response into out, and
// converts non-2xx responses to *Error. A nil out discards the body.
func (b *HTTPBackend) Call(ctx context.Context, method, path string, params, out interface{}) error {
	resp, err := b.call(ctx, method, path, params)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return ParseError(resp, body)
	}

	if isHTML(resp.Header.Get("Content-Type"), body) {
		return &Error{
			Code:       ErrCodeServerError,
			Message:    "unexpected HTML response from Paystack",
			StatusCode: resp.StatusCode,
			RawBody:    body,
		}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return &Error{
			Code:       ErrCodeServerError,
			Message:    fmt.Sprintf("decode response: %s", err),
			StatusCode: resp.StatusCode,
			RawBody:    body,
		}
	}
	return nil
}

// CallRaw returns the live *http.Response. The caller owns resp.Body and must
// close it. Useful for streaming or inspecting headers.
func (b *HTTPBackend) CallRaw(ctx context.Context, method, path string, params interface{}) (*http.Response, error) {
	return b.call(ctx, method, path, params)
}

func (b *HTTPBackend) call(ctx context.Context, method, path string, params interface{}) (*http.Response, error) {
	fullURL, body, idempotencyKey, err := b.buildRequest(method, path, params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s (%s)", userAgentPrefix, sdkVersion, runtime.Version()))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	b.logRequest(method, fullURL)

	return b.client.Do(req)
}

func (b *HTTPBackend) buildRequest(method, path string, params interface{}) (string, io.Reader, string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := b.base + path
	idempotencyKey := extractIdempotencyKey(params)

	if !isWriteMethod(method) {
		if params != nil {
			values, err := query.Values(params)
			if err != nil {
				return "", nil, "", err
			}
			if encoded := values.Encode(); encoded != "" {
				fullURL += "?" + encoded
			}
		}
		return fullURL, nil, idempotencyKey, nil
	}

	if params == nil {
		return fullURL, nil, idempotencyKey, nil
	}
	buf, err := json.Marshal(params)
	if err != nil {
		return "", nil, "", err
	}
	return fullURL, bytes.NewReader(buf), idempotencyKey, nil
}

func (b *HTTPBackend) logRequest(method, url string) {
	switch {
	case b.leveled != nil:
		b.leveled.Debugf("paystack: %s %s", method, url)
	case b.log != nil:
		b.log.Printf("paystack: %s %s", method, url)
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func extractIdempotencyKey(params interface{}) string {
	pc, ok := params.(paramCarrier)
	if !ok {
		return ""
	}
	p := pc.paystackParams()
	if p.IdempotencyKey == nil {
		return ""
	}
	return *p.IdempotencyKey
}

func drainAndClose(rc io.ReadCloser) {
	_, _ = io.Copy(io.Discard, rc)
	_ = rc.Close()
}
