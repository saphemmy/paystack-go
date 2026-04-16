package paystack

import (
	"errors"
	"net/http"
	"strings"
)

// Key is an optional package-level secret key. Setting it is not required —
// callers should prefer passing the key to New — but it exists for parity
// with the Paystack client conventions in other languages.
var Key string

// ErrInvalidKey is returned by New when the supplied secret key does not
// have a recognised Paystack prefix.
var ErrInvalidKey = errors.New("paystack: secret key must start with sk_test_ or sk_live_")

// Option configures a Client built by New. Options are applied in order.
type Option func(*clientOptions)

type clientOptions struct {
	backend    Backend
	httpClient *http.Client
	baseURL    string
	logger     Logger
	leveled    LeveledLogger
}

// WithBackend substitutes the entire Backend. Use it for custom transports,
// recording backends, or to drop in a test double.
func WithBackend(b Backend) Option {
	return func(o *clientOptions) { o.backend = b }
}

// WithHTTPClient overrides the http.Client used by the default HTTPBackend.
// Ignored when WithBackend is also supplied.
func WithHTTPClient(c *http.Client) Option {
	return func(o *clientOptions) { o.httpClient = c }
}

// WithBaseURL points the default HTTPBackend at a non-production host.
// Ignored when WithBackend is also supplied.
func WithBaseURL(url string) Option {
	return func(o *clientOptions) { o.baseURL = url }
}

// WithLogger wires a Printf-style logger into the default HTTPBackend.
// Ignored when WithBackend is also supplied.
func WithLogger(l Logger) Option {
	return func(o *clientOptions) { o.logger = l }
}

// WithLeveledLogger wires a structured logger into the default HTTPBackend.
// Takes precedence over WithLogger when both are set.
func WithLeveledLogger(l LeveledLogger) Option {
	return func(o *clientOptions) { o.leveled = l }
}

// New builds a ClientInterface, validating the secret key up front. A test
// key must start with sk_test_ and a live key with sk_live_; anything else
// is rejected with ErrInvalidKey.
func New(secretKey string, opts ...Option) (ClientInterface, error) {
	if !validKey(secretKey) {
		return nil, ErrInvalidKey
	}
	co := &clientOptions{}
	for _, opt := range opts {
		opt(co)
	}
	return newClient(secretKey, co), nil
}

func validKey(k string) bool {
	return strings.HasPrefix(k, "sk_test_") || strings.HasPrefix(k, "sk_live_")
}

// String returns a pointer to s. Use it to populate optional string fields
// without having to declare a variable.
func String(s string) *string { return &s }

// Int64 returns a pointer to n.
func Int64(n int64) *int64 { return &n }

// Int returns a pointer to n.
func Int(n int) *int { return &n }

// Bool returns a pointer to b.
func Bool(b bool) *bool { return &b }

// Float64 returns a pointer to f.
func Float64(f float64) *float64 { return &f }
