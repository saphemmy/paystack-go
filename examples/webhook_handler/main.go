// A minimal net/http webhook server. Paystack posts events here; the SDK
// verifies the signature and dispatches the event.
//
//	PAYSTACK_SECRET_KEY=sk_test_xxx go run ./examples/webhook_handler
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	paystack "github.com/saphemmy/paystack-go"
)

func main() {
	secret := []byte(os.Getenv("PAYSTACK_SECRET_KEY"))
	if len(secret) == 0 {
		log.Fatal("PAYSTACK_SECRET_KEY is required")
	}

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		ev, err := paystack.ParseWebhook(r, secret)
		if err != nil {
			switch err {
			case paystack.ErrInvalidSignature:
				http.Error(w, "bad signature", http.StatusBadRequest)
			default:
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			return
		}

		switch ev.Type {
		case paystack.EventChargeSuccess:
			var charge struct {
				Reference string `json:"reference"`
				Amount    int64  `json:"amount"`
				Currency  string `json:"currency"`
			}
			_ = json.Unmarshal(ev.Data, &charge)
			log.Printf("charge succeeded: %s %d %s", charge.Reference, charge.Amount, charge.Currency)

		case paystack.EventTransferFailed:
			log.Printf("transfer failed — inspect ev.Data: %s", string(ev.Data))

		default:
			log.Printf("ignoring event: %s", ev.Type)
		}

		w.WriteHeader(http.StatusOK)
	})

	log.Println("listening on :8080 /webhook")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
