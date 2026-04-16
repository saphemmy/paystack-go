// Package paystackfiber wires the Paystack Go SDK into the gofiber framework.
//
// It provides three primitives matching paystackgin and paystackecho:
//
//   - Middleware(client) attaches the SDK client to every fiber.Ctx.
//   - ClientFrom(c) retrieves it in a handler.
//   - WebhookHandler(secret, fn) returns a fiber.Handler that verifies a
//     Paystack webhook signature, parses the event, and dispatches to fn.
package paystackfiber
