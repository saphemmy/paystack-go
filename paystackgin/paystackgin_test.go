package paystackgin_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/paystackgin"
)

func init() { gin.SetMode(gin.TestMode) }

type fakeBackend struct{}

func (fakeBackend) Call(context.Context, string, string, interface{}, interface{}) error {
	return nil
}
func (fakeBackend) CallRaw(context.Context, string, string, interface{}) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

func sign(body, secret []byte) string {
	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestMiddleware_StoresClient(t *testing.T) {
	client, _ := paystack.New("sk_test_x", paystack.WithBackend(fakeBackend{}))
	r := gin.New()
	r.Use(paystackgin.Middleware(client))
	var gotClient paystack.ClientInterface
	r.GET("/ping", func(c *gin.Context) {
		gotClient = paystackgin.ClientFrom(c)
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	if gotClient != client {
		t.Fatal("ClientFrom did not return the Middleware-provided client")
	}
}

func TestClientFrom_ReturnsNilWhenMiddlewareAbsent(t *testing.T) {
	r := gin.New()
	var gotClient paystack.ClientInterface
	ptr := &gotClient
	r.GET("/x", func(c *gin.Context) {
		*ptr = paystackgin.ClientFrom(c)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if gotClient != nil {
		t.Fatalf("expected nil, got %v", gotClient)
	}
}

func TestWebhookHandler_Success(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{"reference":"ref"}}`)
	var captured *paystack.Event

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		captured = ev
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if captured == nil || captured.Type != paystack.EventChargeSuccess {
		t.Fatalf("captured event wrong: %+v", captured)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{}}`)

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("handler should not be called when signature is invalid")
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWebhookHandler_MalformedBody(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":`)

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("handler should not run on malformed body")
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWebhookHandler_HandlerError(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{}}`)

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		return errors.New("downstream broke")
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestWebhookHandler_HandlerReturnsInvalidSignatureError(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{}}`)

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		return paystack.ErrInvalidSignature
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWebhookHandler_OversizedBody(t *testing.T) {
	secret := []byte("sk_test_x")
	big := bytes.Repeat([]byte("x"), paystack.MaxWebhookBodyBytes+1024)

	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("handler should not run on oversized body")
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(big))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(big, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (truncated payload should fail signature)", w.Code)
	}
}

func TestWebhookHandler_PanicsOnNilFn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil fn")
		}
	}()
	paystackgin.WebhookHandler([]byte("s"), nil)
}

func TestWebhookHandler_ReadBodyError(t *testing.T) {
	secret := []byte("sk_test_x")
	r := gin.New()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error { return nil }))

	req := httptest.NewRequest(http.MethodPost, "/webhook", badReader{})
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

type badReader struct{}

func (badReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
