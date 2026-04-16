package paystack

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxWebhookBodyBytes caps the payload ParseWebhook will read from an
// http.Request body. Paystack events are kilobytes at most; anything larger
// is treated as hostile.
const MaxWebhookBodyBytes = 1 << 20 // 1 MiB

// WebhookSignatureHeader is the HTTP header Paystack uses to transmit the
// HMAC signature of the webhook body.
const WebhookSignatureHeader = "X-Paystack-Signature"

// ErrInvalidSignature is returned by ParseWebhook when the body's HMAC does
// not match the header.
var ErrInvalidSignature = errors.New("paystack: invalid webhook signature")

// EventType is a Paystack webhook event name.
type EventType string

// The full set of Paystack webhook events supported by this SDK.
// Callers should switch on Event.Type and unmarshal Event.Data into the
// concrete payload they expect.
const (
	EventChargeSuccess             EventType = "charge.success"
	EventChargeDisputeCreate       EventType = "charge.dispute.create"
	EventChargeDisputeRemind       EventType = "charge.dispute.remind"
	EventChargeDisputeResolve      EventType = "charge.dispute.resolve"
	EventTransferSuccess           EventType = "transfer.success"
	EventTransferFailed            EventType = "transfer.failed"
	EventTransferReversed          EventType = "transfer.reversed"
	EventSubscriptionCreate        EventType = "subscription.create"
	EventSubscriptionDisable       EventType = "subscription.disable"
	EventSubscriptionNotRenew      EventType = "subscription.not_renew"
	EventSubscriptionExpiringCards EventType = "subscription.expiring_cards"
	EventInvoiceCreate             EventType = "invoice.create"
	EventInvoiceUpdate             EventType = "invoice.update"
	EventInvoicePaymentFailed      EventType = "invoice.payment_failed"
	EventPaymentRequestPending     EventType = "paymentrequest.pending"
	EventPaymentRequestSuccess     EventType = "paymentrequest.success"
	EventCustomerIdentification    EventType = "customeridentification.success"
	EventDedicatedAccountAssign    EventType = "dedicatedaccount.assign.success"
	EventRefundProcessed           EventType = "refund.processed"
	EventRefundFailed              EventType = "refund.failed"
)

// Event is the wire shape of a Paystack webhook. Data is the raw JSON of the
// event's `data` object; callers unmarshal it into the concrete type they
// expect. No typed unions — keeps the API stable as Paystack adds fields.
type Event struct {
	Type EventType       `json:"event"`
	Data json.RawMessage `json:"data"`
}

// Verify reports whether sig matches the HMAC-SHA512 of body under secret.
// Uses hmac.Equal to prevent timing attacks. Accepts hex-encoded signatures
// in either case.
func Verify(body []byte, sig string, secret []byte) bool {
	if len(sig) == 0 || len(secret) == 0 {
		return false
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	return hmac.Equal(got, mac.Sum(nil))
}

// ParseEvent decodes a webhook body into an Event. The caller must have
// already verified the signature with Verify.
func ParseEvent(body []byte) (*Event, error) {
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("paystack: parse event: %w", err)
	}
	if ev.Type == "" {
		return nil, errors.New("paystack: event payload missing `event` field")
	}
	return &ev, nil
}

// ParseWebhook reads, size-limits, verifies, and parses a webhook request in
// one call. It is the recommended entry point for http.Handler-based servers
// that aren't using one of the framework integration packages.
//
// The caller is responsible for writing the HTTP response. ParseWebhook never
// closes r.Body — that remains the caller's contract with the net/http server.
func ParseWebhook(r *http.Request, secret []byte) (*Event, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxWebhookBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("paystack: read webhook body: %w", err)
	}
	if !Verify(body, r.Header.Get(WebhookSignatureHeader), secret) {
		return nil, ErrInvalidSignature
	}
	return ParseEvent(body)
}
