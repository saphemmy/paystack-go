package paystackecho_test

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

	"github.com/labstack/echo/v4"
	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/paystackecho"
)

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
	e := echo.New()
	e.Use(paystackecho.Middleware(client))
	var gotClient paystack.ClientInterface
	e.GET("/ping", func(c echo.Context) error {
		gotClient = paystackecho.ClientFrom(c)
		return c.NoContent(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	if gotClient != client {
		t.Fatal("ClientFrom mismatch")
	}
}

func TestClientFrom_ReturnsNilWhenMiddlewareAbsent(t *testing.T) {
	e := echo.New()
	var gotClient paystack.ClientInterface
	ptr := &gotClient
	e.GET("/x", func(c echo.Context) error {
		*ptr = paystackecho.ClientFrom(c)
		return c.NoContent(http.StatusOK)
	})
	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if gotClient != nil {
		t.Fatalf("expected nil, got %v", gotClient)
	}
}

func TestWebhookHandler_Success(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{"reference":"ref"}}`)
	var captured *paystack.Event

	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error {
		captured = ev
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if captured == nil || captured.Type != paystack.EventChargeSuccess {
		t.Fatalf("captured = %+v", captured)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler([]byte("s"), func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	body := []byte(`{"event":"charge.success","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookHandler_MalformedBody(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":`)
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookHandler_HandlerError(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":"charge.success","data":{}}`)
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error {
		return errors.New("broke")
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookHandler_HandlerReturnsInvalidSignatureError(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":"charge.success","data":{}}`)
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error {
		return paystack.ErrInvalidSignature
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookHandler_OversizedBody(t *testing.T) {
	secret := []byte("s")
	big := bytes.Repeat([]byte("x"), paystack.MaxWebhookBodyBytes+1024)
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(big))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(big, secret))
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWebhookHandler_PanicsOnNilFn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	paystackecho.WebhookHandler([]byte("s"), nil)
}

func TestWebhookHandler_ReadBodyError(t *testing.T) {
	secret := []byte("s")
	e := echo.New()
	e.POST("/webhook", paystackecho.WebhookHandler(secret, func(ev *paystack.Event) error { return nil }))
	req := httptest.NewRequest(http.MethodPost, "/webhook", badReader{})
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
}

type badReader struct{}

func (badReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
