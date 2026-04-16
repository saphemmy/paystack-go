package paystackfiber

import (
	"github.com/gofiber/fiber/v2"
	paystack "github.com/saphemmy/paystack-go"
)

const contextKey = "paystackfiber.client"

// Middleware returns a fiber.Handler that stashes client on the context.
func Middleware(client paystack.ClientInterface) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(contextKey, client)
		return c.Next()
	}
}

// ClientFrom returns the paystack client attached by Middleware, or nil.
func ClientFrom(c *fiber.Ctx) paystack.ClientInterface {
	v := c.Locals(contextKey)
	if v == nil {
		return nil
	}
	client, _ := v.(paystack.ClientInterface)
	return client
}
