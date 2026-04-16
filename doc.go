// Package paystack is the Go SDK for the Paystack payments API.
//
// It is a lean, transport-faithful client. The SDK represents the HTTP API
// exactly; retry strategies, currency conversion, and framework lifecycle
// belong to the caller or to one of the framework integration packages
// (paystackgin, paystackfiber, paystackecho).
//
// # Getting started
//
// Construct a client with your secret key. Validation happens up front —
// keys must begin with sk_test_ or sk_live_.
//
//	client, err := paystack.New("sk_test_xxx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	tx, err := client.Transaction().Initialize(ctx, &paystack.TransactionInitializeParams{
//	    Email:  "customer@example.com",
//	    Amount: 500000, // kobo: 5,000.00 NGN
//	})
//
// # Amounts
//
// All monetary values are int64 in kobo (1 NGN = 100 kobo). The SDK never
// performs currency conversion.
//
// # Errors
//
// Every non-2xx response is returned as a *Error. Callers branch on Code:
//
//	var pErr *paystack.Error
//	if errors.As(err, &pErr) {
//	    switch pErr.Code {
//	    case paystack.ErrCodeRateLimited:
//	        time.Sleep(pErr.RetryAfter)
//	    case paystack.ErrCodeInvalidRequest:
//	        for field, msg := range pErr.Fields {
//	            log.Printf("%s: %s", field, msg)
//	        }
//	    }
//	}
//
// # Extensibility
//
// The SDK exposes stable interfaces — ClientInterface, Backend, the per-resource
// service interfaces (Transactor, Customeror, Planner, Subscriber, Transferor,
// Charger, Refunder), Logger, and LeveledLogger — so callers can wrap,
// instrument, or mock any layer without forking the SDK.
//
// # Webhooks
//
// Verify uses hmac.Equal for constant-time comparison; ParseEvent returns a
// raw Event whose Data field the caller unmarshals into the concrete type.
package paystack
