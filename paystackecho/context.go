package paystackecho

import (
	"github.com/labstack/echo/v4"
	paystack "github.com/saphemmy/paystack-go"
)

const contextKey = "paystackecho.client"

// Middleware returns an echo.MiddlewareFunc that stashes client on the
// context.
func Middleware(client paystack.ClientInterface) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(contextKey, client)
			return next(c)
		}
	}
}

// ClientFrom returns the paystack client attached by Middleware, or nil.
func ClientFrom(c echo.Context) paystack.ClientInterface {
	v := c.Get(contextKey)
	if v == nil {
		return nil
	}
	client, _ := v.(paystack.ClientInterface)
	return client
}
