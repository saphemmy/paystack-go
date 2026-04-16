package paystackecho

import (
	"errors"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	paystack "github.com/saphemmy/paystack-go"
)

// WebhookHandler returns an echo.HandlerFunc that verifies the signature,
// parses the event, and dispatches it to fn. See paystackgin for status codes.
func WebhookHandler(secret []byte, fn func(*paystack.Event) error) echo.HandlerFunc {
	if fn == nil {
		panic("paystackecho: WebhookHandler requires a non-nil handler")
	}
	return func(c echo.Context) error {
		body, err := io.ReadAll(io.LimitReader(c.Request().Body, paystack.MaxWebhookBodyBytes))
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}
		if !paystack.Verify(body, c.Request().Header.Get(paystack.WebhookSignatureHeader), secret) {
			return c.NoContent(http.StatusBadRequest)
		}
		ev, err := paystack.ParseEvent(body)
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}
		if err := fn(ev); err != nil {
			if errors.Is(err, paystack.ErrInvalidSignature) {
				return c.NoContent(http.StatusBadRequest)
			}
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusOK)
	}
}
