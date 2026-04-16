// Same webhook server as examples/webhook_handler, but routed through the
// paystackgin adapter.
//
//	PAYSTACK_SECRET_KEY=sk_test_xxx go run ./examples/webhook_with_gin
package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	paystack "github.com/saphemmy/paystack-go"
	"github.com/saphemmy/paystack-go/paystackgin"
)

func main() {
	secret := []byte(os.Getenv("PAYSTACK_SECRET_KEY"))
	if len(secret) == 0 {
		log.Fatal("PAYSTACK_SECRET_KEY is required")
	}

	r := gin.Default()
	r.POST("/webhook", paystackgin.WebhookHandler(secret, func(ev *paystack.Event) error {
		switch ev.Type {
		case paystack.EventChargeSuccess:
			var c struct {
				Reference string `json:"reference"`
				Amount    int64  `json:"amount"`
			}
			if err := json.Unmarshal(ev.Data, &c); err != nil {
				return err
			}
			log.Printf("charge succeeded: %s %d", c.Reference, c.Amount)
		default:
			log.Printf("ignoring event: %s", ev.Type)
		}
		return nil
	}))

	log.Fatal(r.Run(":8080"))
}
