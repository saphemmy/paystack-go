# Examples

Runnable sample programs that exercise `paystack-go` end-to-end. Each
sub-directory is its own `package main`; run with `go run`:

```bash
# Requires a test secret key in the environment.
export PAYSTACK_SECRET_KEY=sk_test_xxx

go run ./examples/initialize_transaction
go run ./examples/verify_transaction
go run ./examples/webhook_handler
go run ./examples/webhook_with_gin
```

The webhook examples start a local HTTP server on `:8080`. Point your
Paystack dashboard's webhook URL at `http://<your-host>:8080/webhook`
to exercise them end to end, or post signed payloads manually:

```bash
BODY='{"event":"charge.success","data":{"reference":"ref_123"}}'
SIG=$(printf '%s' "$BODY" | openssl dgst -sha512 -hmac "$PAYSTACK_SECRET_KEY" | awk '{print $2}')
curl -X POST http://localhost:8080/webhook \
  -H "X-Paystack-Signature: $SIG" \
  -d "$BODY"
```

## Index

| Folder                  | What it shows                                         |
| ----------------------- | ----------------------------------------------------- |
| `initialize_transaction`| Build a checkout URL for a new payment.               |
| `verify_transaction`    | Look up the final status of a payment by reference.   |
| `webhook_handler`       | Plain `net/http` signature-verifying webhook server.  |
| `webhook_with_gin`      | Same thing with `paystackgin.WebhookHandler`.         |
