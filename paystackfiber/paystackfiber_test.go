package paystackfiber_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/paystackfiber"
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

func newApp() *fiber.App {
	return fiber.New(fiber.Config{DisableStartupMessage: true})
}

func runReq(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

func TestMiddleware_StoresClient(t *testing.T) {
	client, _ := paystack.New("sk_test_x", paystack.WithBackend(fakeBackend{}))
	var gotClient paystack.ClientInterface
	app := newApp()
	app.Use(paystackfiber.Middleware(client))
	app.Get("/ping", func(c *fiber.Ctx) error {
		gotClient = paystackfiber.ClientFrom(c)
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp := runReq(t, app, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if gotClient != client {
		t.Fatal("ClientFrom mismatch")
	}
}

func TestClientFrom_ReturnsNilWhenMiddlewareAbsent(t *testing.T) {
	app := newApp()
	var gotClient paystack.ClientInterface
	ptr := &gotClient
	app.Get("/x", func(c *fiber.Ctx) error {
		*ptr = paystackfiber.ClientFrom(c)
		return c.SendStatus(fiber.StatusOK)
	})
	_ = runReq(t, app, httptest.NewRequest(http.MethodGet, "/x", nil))
	if gotClient != nil {
		t.Fatalf("expected nil, got %v", gotClient)
	}
}

func TestWebhookHandler_Success(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{"reference":"ref"}}`)
	var captured *paystack.Event

	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler(secret, func(ev *paystack.Event) error {
		captured = ev
		return nil
	}))

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if captured == nil || captured.Type != paystack.EventChargeSuccess {
		t.Fatalf("captured = %+v", captured)
	}
}

func TestWebhookHandler_InvalidSignature(t *testing.T) {
	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler([]byte("s"), func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	body := []byte(`{"event":"charge.success","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestWebhookHandler_MalformedBody(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":`)
	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestWebhookHandler_HandlerError(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":"charge.success","data":{}}`)
	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler(secret, func(ev *paystack.Event) error {
		return errors.New("broke")
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestWebhookHandler_HandlerReturnsInvalidSignatureError(t *testing.T) {
	secret := []byte("s")
	body := []byte(`{"event":"charge.success","data":{}}`)
	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler(secret, func(ev *paystack.Event) error {
		return paystack.ErrInvalidSignature
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(body, secret))
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestWebhookHandler_OversizedBody(t *testing.T) {
	secret := []byte("s")
	big := bytes.Repeat([]byte("x"), paystack.MaxWebhookBodyBytes+1024)
	app := newApp()
	app.Post("/webhook", paystackfiber.WebhookHandler(secret, func(ev *paystack.Event) error {
		t.Fatal("should not run")
		return nil
	}))
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(big))
	req.Header.Set(paystack.WebhookSignatureHeader, sign(big, secret))
	resp := runReq(t, app, req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestWebhookHandler_PanicsOnNilFn(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	paystackfiber.WebhookHandler([]byte("s"), nil)
}
