// Verify a transaction by reference and print its final status.
//
//	PAYSTACK_SECRET_KEY=sk_test_xxx \
//	PAYSTACK_REFERENCE=ref_123 \
//	  go run ./examples/verify_transaction
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

func main() {
	key := os.Getenv("PAYSTACK_SECRET_KEY")
	ref := os.Getenv("PAYSTACK_REFERENCE")
	if key == "" || ref == "" {
		log.Fatal("PAYSTACK_SECRET_KEY and PAYSTACK_REFERENCE are required")
	}

	client, err := paystack.New(key)
	if err != nil {
		log.Fatalf("paystack.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := client.Transaction().Verify(ctx, ref)
	if err != nil {
		var pErr *paystack.Error
		if errors.As(err, &pErr) && pErr.Code == paystack.ErrCodeRateLimited {
			log.Fatalf("rate limited; retry after %s", pErr.RetryAfter)
		}
		log.Fatalf("Verify: %v", err)
	}

	fmt.Printf("reference: %s\n", tx.Reference)
	fmt.Printf("status:    %s\n", tx.Status)
	fmt.Printf("amount:    %d %s\n", tx.Amount, tx.Currency)
	if tx.Customer != nil {
		fmt.Printf("customer:  %s\n", tx.Customer.Email)
	}
}
