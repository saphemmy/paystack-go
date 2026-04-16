# paystack-go

[![CI](https://github.com/saphemmy/paystack-go/actions/workflows/ci.yml/badge.svg)](https://github.com/saphemmy/paystack-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/saphemmy/paystack-go.svg)](https://pkg.go.dev/github.com/saphemmy/paystack-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/saphemmy/paystack-go)](https://goreportcard.com/report/github.com/saphemmy/paystack-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

A production-grade Go SDK for [Paystack](https://paystack.com) — the African
payments gateway that merchants use to accept card, bank, USSD, mobile-money,
and transfer payments across Nigeria, Ghana, South Africa, Kenya, and
Côte d'Ivoire.

Built against the [Paystack REST API](https://paystack.com/docs/api/) for
teams that need to embed Paystack inside a larger platform — billing
systems, marketplaces, banking-as-a-service, ERPs, fintech cores — where
correctness, testability, observability, and interface stability matter
more than convenience.

**Useful links:**

- 🏠 Paystack homepage — <https://paystack.com>
- 📚 API reference — <https://paystack.com/docs/api/>
- 🔑 Get API keys — <https://dashboard.paystack.com/#/settings/developers>
- 📡 Webhooks guide — <https://paystack.com/docs/payments/webhooks/>

> **Status:** pre-1.0. Interfaces are stable on `main` but minor versions may
> introduce additive changes to request/response structs. See
> [Versioning](#versioning) for the compatibility policy.

## Why this SDK

Designed for enterprise systems rather than one-off scripts. Every design
decision maps to a concrete operational concern:

| Concern                  | How it's addressed                                                                                                 |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------ |
| **Testability**          | Every resource sits behind a `Transactor` / `Customeror` / … interface. Swap the `Backend` in tests — no live HTTP. |
| **Idempotency**          | `Params.IdempotencyKey` is forwarded as a header on every write, so safe to retry across crashes.                   |
| **Observability**        | `Logger` and `LeveledLogger` seams route SDK logs into your platform's logging stack (zap, slog, logrus, zerolog).  |
| **Auditable errors**     | One `*Error` type with stable `Code` constants (`ErrCodeInvalidKey`, `ErrCodeRateLimited`, …), populated `Fields` on validation errors, and parsed `RetryAfter`. Works with `errors.As`. |
| **No hidden retries**    | The SDK never retries. You own the policy — pair it with your existing circuit-breaker / retry library.             |
| **No currency surprises**| Amounts are `int64` in kobo end-to-end. No silent conversion, no float.                                            |
| **Constant-time crypto** | Webhook `Verify` uses `hmac.Equal`; body reads are `LimitReader`-capped to defend against oversized payloads.       |
| **Stable contract**      | `ClientInterface`, `Backend`, service interfaces, and `Event` never break without a major bump — safe to depend on. |
| **Framework-agnostic**   | Core SDK has one production dependency (`go-querystring`). Gin/Fiber/Echo adapters ship as separate modules so you only pay for what you use. |
| **Context-first**        | Every call takes `context.Context` — timeouts and cancellation propagate through your request graph.                |
| **Compliance surface**   | SDK never stores card data or keys at rest. Everything is stateless; caller owns the vault.                         |

## Design

- `paystack-go` is the transport- and API-faithful core.
- `paystackgin`, `paystackfiber`, `paystackecho` are thin framework adapters
  that import this package's **interfaces**, never its concrete types.

The SDK represents the Paystack API faithfully. Retry strategies, currency
conversion, framework lifecycle, and business logic belong to the caller or
an integration package — not the SDK.

## Installation

```bash
go get github.com/saphemmy/paystack-go
```

Go 1.22 or later.

## Quick start

```go
package main

import (
    "context"
    "log"

    paystack "github.com/saphemmy/paystack-go"
)

func main() {
    client, err := paystack.New("sk_test_xxx")
    if err != nil {
        log.Fatal(err)
    }

    tx, err := client.Transaction().Initialize(context.Background(), &paystack.TransactionInitializeParams{
        Email:  "customer@example.com",
        Amount: 500000, // kobo — 5,000.00 NGN
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Println(tx.AuthorizationURL)
}
```

Amounts are in **kobo** (1 NGN = 100 kobo). This is never silently converted.

## Framework integrations

| Framework | Package                                                |
| --------- | ------------------------------------------------------ |
| Gin       | `github.com/saphemmy/paystack-go/paystackgin`          |
| Fiber     | `github.com/saphemmy/paystack-go/paystackfiber`        |
| Echo      | `github.com/saphemmy/paystack-go/paystackecho`         |

Each is a separate Go module, importable independently.

## Error handling

Every non-2xx response is returned as a `*paystack.Error`. Callers switch on
`err.Code`:

```go
tx, err := client.Transaction().Verify(ctx, reference)
var pErr *paystack.Error
if errors.As(err, &pErr) {
    switch pErr.Code {
    case paystack.ErrCodeRateLimited:
        time.Sleep(pErr.RetryAfter)
        // retry — the SDK never retries for you
    case paystack.ErrCodeInvalidRequest:
        for field, msg := range pErr.Fields {
            log.Printf("%s: %s", field, msg)
        }
    }
}
```

## Webhooks

```go
import "github.com/saphemmy/paystack-go"

func handler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
    if !paystack.Verify(body, r.Header.Get("x-paystack-signature"), secret) {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }
    event, err := paystack.ParseEvent(body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    switch event.Type {
    case paystack.EventChargeSuccess:
        var charge paystack.Charge
        _ = json.Unmarshal(event.Data, &charge)
        // ...
    }
    w.WriteHeader(http.StatusOK)
}
```

Framework packages ship a `WebhookHandler` that wraps verification, body size
limiting, and parsing.

## Testing

The SDK exposes a `Backend` interface so you never need to hit the live API in
tests. `internal/testutil` ships `MockBackend` and `FixtureBackend` — these
are used by the SDK's own tests and are not part of the public API.

For downstream tests, swap the `Backend` via `paystack.WithBackend(...)`.

Run the suite:

```bash
go test ./...
go test -race ./...
go test -cover ./...
```

Integration tests against the Paystack sandbox live behind the `integration`
build tag:

```bash
PAYSTACK_TEST_KEY=sk_test_xxx go test -tags=integration ./...
```

## Versioning

- **Stable surface** — `ClientInterface`, `Backend`, service interfaces,
  `Event`, `Logger`, `LeveledLogger`. Breaking changes require a major bump.
- **Internal** — concrete service structs and `HTTPBackend` internals may
  change between minor versions.
- **Request/response structs** — additive in minor versions. Fields are never
  removed or renamed without a major bump.

See [CHANGELOG.md](./CHANGELOG.md) for release history.

## Contributing

Pull requests are welcome. Please read
[CONTRIBUTING.md](./CONTRIBUTING.md) and our
[Code of Conduct](./CODE_OF_CONDUCT.md) first.

To report a security issue, see [SECURITY.md](./SECURITY.md).

## License

[MIT](./LICENSE) © Oluwafemi Sosami
