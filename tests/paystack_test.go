package paystack_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

func TestNew_ValidatesKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"test key", "sk_test_abc", false},
		{"live key", "sk_live_abc", false},
		{"public key rejected", "pk_test_abc", true},
		{"empty", "", true},
		{"garbage", "not-a-key", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, err := paystack.New(tc.key)
			if tc.wantErr {
				if !errors.Is(err, paystack.ErrInvalidKey) {
					t.Fatalf("want ErrInvalidKey, got %v", err)
				}
				if c != nil {
					t.Fatal("client should be nil on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if c == nil {
				t.Fatal("client should not be nil")
			}
		})
	}
}

func TestNew_AppliesOptions(t *testing.T) {
	// Behavioural test: run the client against a local server to confirm
	// WithBaseURL and WithHTTPClient are honoured end-to-end.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"message":"ok","data":{"authorization_url":"u","access_code":"a","reference":"r"}}`))
	}))
	defer srv.Close()

	c, err := paystack.New("sk_test_x", paystack.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	data, err := c.Transaction().Initialize(context.Background(), &paystack.TransactionInitializeParams{Email: "e@f.g", Amount: 1})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if data.Reference != "r" {
		t.Fatalf("Reference = %q", data.Reference)
	}
}

func TestNew_WithLoggerEmitsOnRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{}}`))
	}))
	defer srv.Close()

	l := &recordingLogger{}
	c, _ := paystack.New("sk_test_x", paystack.WithBaseURL(srv.URL), paystack.WithLogger(l))
	_, _ = c.Customer().Fetch(context.Background(), "CUS_x")
	if !strings.Contains(l.buf.String(), "GET") {
		t.Fatalf("expected logger to record GET, got %q", l.buf.String())
	}
}

func TestNew_WithLeveledLoggerEmitsOnRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":true,"data":{}}`))
	}))
	defer srv.Close()

	l := &recordingLeveled{}
	c, _ := paystack.New("sk_test_x", paystack.WithBaseURL(srv.URL), paystack.WithLeveledLogger(l))
	_, _ = c.Customer().Fetch(context.Background(), "CUS_x")
	if !strings.Contains(l.debug.String(), "GET") {
		t.Fatalf("expected leveled logger to record GET, got %q", l.debug.String())
	}
}

func TestNew_WithBackendOverrides(t *testing.T) {
	backend := &nopBackend{}
	c, err := paystack.New("sk_test_x", paystack.WithBackend(backend))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.Backend() != backend {
		t.Fatal("WithBackend did not override")
	}
}

func TestNew_WithHTTPClientAppliesCustomTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// intentionally slow — client timeout should fire first.
		<-r.Context().Done()
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	c, _ := paystack.New("sk_test_x", paystack.WithBaseURL(srv.URL), paystack.WithHTTPClient(client))
	_, err := c.Customer().Fetch(context.Background(), "CUS_x")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPointerHelpers(t *testing.T) {
	if *paystack.String("x") != "x" {
		t.Fatal("String")
	}
	if *paystack.Int64(7) != 7 {
		t.Fatal("Int64")
	}
	if *paystack.Int(8) != 8 {
		t.Fatal("Int")
	}
	if *paystack.Bool(true) != true {
		t.Fatal("Bool")
	}
	if *paystack.Float64(1.5) != 1.5 {
		t.Fatal("Float64")
	}
}
