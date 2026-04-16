package paystackfiber

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	paystack "github.com/saphemmy/paystack-go"
)

// WebhookHandler returns a fiber.Handler that verifies the signature, parses
// the event, and dispatches it to fn. See the corresponding doc in
// paystackgin for status codes; this handler behaves identically.
func WebhookHandler(secret []byte, fn func(*paystack.Event) error) fiber.Handler {
	if fn == nil {
		panic("paystackfiber: WebhookHandler requires a non-nil handler")
	}
	return func(c *fiber.Ctx) error {
		body := c.Body()
		if len(body) > paystack.MaxWebhookBodyBytes {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		if !paystack.Verify(body, c.Get(paystack.WebhookSignatureHeader), secret) {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		ev, err := paystack.ParseEvent(body)
		if err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		if err := fn(ev); err != nil {
			if errors.Is(err, paystack.ErrInvalidSignature) {
				return c.SendStatus(fiber.StatusBadRequest)
			}
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusOK)
	}
}
