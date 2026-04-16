// Package paystackgin wires the Paystack Go SDK into the gin-gonic framework.
//
// It provides three primitives that mirror paystackfiber and paystackecho:
//
//   - Middleware(client) attaches the SDK client to every gin.Context.
//   - ClientFrom(c) retrieves it in a handler.
//   - WebhookHandler(secret, fn) returns a gin.HandlerFunc that verifies a
//     Paystack webhook signature, parses the event, and dispatches to fn.
//
// The package depends on the stable interfaces exported by paystack-go
// (ClientInterface, Event, ParseWebhook) — never the concrete Client.
package paystackgin
