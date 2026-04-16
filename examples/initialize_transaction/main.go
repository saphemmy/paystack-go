// Initialize a transaction and print the checkout URL Paystack returns.
//
//	PAYSTACK_SECRET_KEY=sk_test_xxx go run ./examples/initialize_transaction
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	paystack "github.com/saphemmy/paystack-go"
)

func main() {
	key := os.Getenv("PAYSTACK_SECRET_KEY")
	if key == "" {
		log.Fatal("PAYSTACK_SECRET_KEY is required")
	}

	client, err := paystack.New(key)
	if err != nil {
		log.Fatalf("paystack.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := client.Transaction().Initialize(ctx, &paystack.TransactionInitializeParams{
		Email:  "customer@example.com",
		Amount: 500000, // kobo → 5,000.00 NGN
	})
	if err != nil {
		log.Fatalf("Initialize: %v", err)
	}

	fmt.Printf("reference:          %s\n", tx.Reference)
	fmt.Printf("authorization URL:  %s\n", tx.AuthorizationURL)
	fmt.Printf("access code:        %s\n", tx.AccessCode)
}
