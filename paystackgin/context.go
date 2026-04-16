package paystackgin

import (
	"github.com/gin-gonic/gin"
	paystack "github.com/saphemmy/paystack-go"
)

// contextKey is the gin.Context key under which Middleware stores the SDK
// client. It is unexported so callers must go through ClientFrom.
const contextKey = "paystackgin.client"

// Middleware returns a gin.HandlerFunc that stashes client on the context so
// downstream handlers can fetch it with ClientFrom.
func Middleware(client paystack.ClientInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(contextKey, client)
		c.Next()
	}
}

// ClientFrom returns the paystack client attached by Middleware. Returns nil
// when Middleware has not been installed.
func ClientFrom(c *gin.Context) paystack.ClientInterface {
	v, ok := c.Get(contextKey)
	if !ok {
		return nil
	}
	client, _ := v.(paystack.ClientInterface)
	return client
}
