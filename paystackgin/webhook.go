package paystackgin

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	paystack "github.com/saphemmy/paystack-go"
)

// WebhookHandler returns a gin.HandlerFunc that verifies the X-Paystack-
// Signature header, parses the event, and dispatches it to fn. It enforces
// the same body-size cap as paystack.ParseWebhook.
//
// Status codes:
//
//   - 200 OK — fn returned nil.
//   - 400 Bad Request — signature missing or invalid, or body malformed.
//   - 500 Internal Server Error — fn returned a non-nil error.
func WebhookHandler(secret []byte, fn func(*paystack.Event) error) gin.HandlerFunc {
	if fn == nil {
		panic("paystackgin: WebhookHandler requires a non-nil handler")
	}
	return func(c *gin.Context) {
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, paystack.MaxWebhookBodyBytes))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if !paystack.Verify(body, c.GetHeader(paystack.WebhookSignatureHeader), secret) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ev, err := paystack.ParseEvent(body)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if err := fn(ev); err != nil {
			if errors.Is(err, paystack.ErrInvalidSignature) {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
	}
}
