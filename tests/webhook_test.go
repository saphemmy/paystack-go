package paystack_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	paystack "github.com/saphemmy/paystack-go"
)

func signPayload(t testing.TB, body, secret []byte) string {
	t.Helper()
	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerify(t *testing.T) {
	secret := []byte("sk_test_supersecret")
	body := []byte(`{"event":"charge.success","data":{"amount":100}}`)
	good := signPayload(t, body, secret)

	tests := []struct {
		name   string
		body   []byte
		sig    string
		secret []byte
		want   bool
	}{
		{"valid signature", body, good, secret, true},
		{"tampered body", []byte(`{"event":"charge.success","data":{"amount":200}}`), good, secret, false},
		{"tampered signature", body, flipHex(good), secret, false},
		{"wrong secret", body, good, []byte("other"), false},
		{"empty signature", body, "", secret, false},
		{"empty secret", body, good, nil, false},
		{"non-hex signature", body, "not-hex-!!", secret, false},
		{"uppercase signature accepted", body, strings.ToUpper(good), secret, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := paystack.Verify(tc.body, tc.sig, tc.secret); got != tc.want {
				t.Fatalf("Verify = %v, want %v", got, tc.want)
			}
		})
	}
}

func flipHex(s string) string {
	if len(s) == 0 {
		return s
	}
	b := []byte(s)
	if b[0] == 'a' {
		b[0] = 'b'
	} else {
		b[0] = 'a'
	}
	return string(b)
}

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		wantErr bool
		want    paystack.EventType
	}{
		{"charge.success", []byte(`{"event":"charge.success","data":{}}`), false, paystack.EventChargeSuccess},
		{"transfer.failed", []byte(`{"event":"transfer.failed","data":{}}`), false, paystack.EventTransferFailed},
		{"malformed json", []byte(`{"event":`), true, ""},
		{"missing event field", []byte(`{"data":{}}`), true, ""},
		{"empty body", []byte(``), true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := paystack.ParseEvent(tc.body)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEvent: %v", err)
			}
			if ev.Type != tc.want {
				t.Fatalf("Type = %q", ev.Type)
			}
		})
	}
}

func TestParseEvent_DataKeptAsRaw(t *testing.T) {
	body := []byte(`{"event":"charge.success","data":{"reference":"ref_1","amount":5000}}`)
	ev, err := paystack.ParseEvent(body)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	var payload struct {
		Reference string `json:"reference"`
		Amount    int64  `json:"amount"`
	}
	if err := json.Unmarshal(ev.Data, &payload); err != nil {
		t.Fatalf("Unmarshal data: %v", err)
	}
	if payload.Reference != "ref_1" || payload.Amount != 5000 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestEventTypeConstants(t *testing.T) {
	wants := map[paystack.EventType]string{
		paystack.EventChargeSuccess:         "charge.success",
		paystack.EventTransferSuccess:       "transfer.success",
		paystack.EventTransferFailed:        "transfer.failed",
		paystack.EventTransferReversed:      "transfer.reversed",
		paystack.EventSubscriptionCreate:    "subscription.create",
		paystack.EventSubscriptionDisable:   "subscription.disable",
		paystack.EventInvoiceCreate:         "invoice.create",
		paystack.EventInvoiceUpdate:         "invoice.update",
		paystack.EventPaymentRequestPending: "paymentrequest.pending",
		paystack.EventPaymentRequestSuccess: "paymentrequest.success",
		paystack.EventRefundProcessed:       "refund.processed",
	}
	for got, want := range wants {
		if string(got) != want {
			t.Fatalf("EventType(%q) != %q", got, want)
		}
	}
}

func TestParseWebhook_Success(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{"reference":"ref"}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, signPayload(t, body, secret))

	ev, err := paystack.ParseWebhook(req, secret)
	if err != nil {
		t.Fatalf("ParseWebhook: %v", err)
	}
	if ev.Type != paystack.EventChargeSuccess {
		t.Fatalf("Type = %q", ev.Type)
	}
}

func TestParseWebhook_InvalidSignature(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":"charge.success","data":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, "deadbeef")

	_, err := paystack.ParseWebhook(req, secret)
	if !errors.Is(err, paystack.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestParseWebhook_MalformedBodyAfterValidSignature(t *testing.T) {
	secret := []byte("sk_test_x")
	body := []byte(`{"event":`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set(paystack.WebhookSignatureHeader, signPayload(t, body, secret))

	_, err := paystack.ParseWebhook(req, secret)
	if err == nil {
		t.Fatal("expected parse error after valid signature")
	}
	if errors.Is(err, paystack.ErrInvalidSignature) {
		t.Fatal("should not report signature error on malformed body")
	}
}

func TestParseWebhook_OversizedBodyRejected(t *testing.T) {
	secret := []byte("sk_test_x")
	large := bytes.Repeat([]byte("A"), paystack.MaxWebhookBodyBytes+1024)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(large))
	req.Header.Set(paystack.WebhookSignatureHeader, signPayload(t, large, secret))

	_, err := paystack.ParseWebhook(req, secret)
	if !errors.Is(err, paystack.ErrInvalidSignature) {
		t.Fatalf("expected signature failure on truncated body, got %v", err)
	}
}

func TestParseWebhook_ReadBodyError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/webhook", errReader{})
	_, err := paystack.ParseWebhook(req, []byte("secret"))
	if err == nil {
		t.Fatal("expected read error")
	}
	if errors.Is(err, paystack.ErrInvalidSignature) {
		t.Fatal("read errors must not be masked as signature errors")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
