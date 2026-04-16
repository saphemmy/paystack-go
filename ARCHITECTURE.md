# Architecture — Paystack Go SDK

## What This Is

The core Paystack Go SDK. Designed to be:
- Used standalone in any Go application
- Extended by framework integration packages without forking
- Composed into larger systems via stable interfaces
- Fully tested at every layer without live HTTP

---

## Repository Layout

```
paystack-go/
├── ARCHITECTURE.md
├── go.mod
│
├── paystack.go                     # Package key, New(), pointer helpers
├── backend.go                      # Backend interface + HTTPBackend
├── client.go                       # Client struct + ClientInterface
├── params.go                       # Params, ListParams, ListResponse[T], Meta
├── errors.go                       # *Error, ErrorCode constants
├── time.go                         # Custom Time type, multi-layout JSON
├── webhook.go                      # Verify(), ParseEvent(), Event
├── logger.go                       # Logger + LeveledLogger interfaces
│
├── transaction.go                  # TransactionService + Transactor interface
├── customer.go                     # CustomerService + Customeror interface
├── plan.go                         # PlanService + Planner interface
├── subscription.go                 # SubscriptionService + Subscriber interface
├── transfer.go                     # TransferService + Transferor interface
├── charge.go                       # ChargeService + Charger interface
├── refund.go                       # RefundService + Refunder interface
│
├── internal/
│   └── testutil/
│       ├── backend.go              # MockBackend, FixtureBackend
│       └── fixtures.go             # LoadFixture(t, name)
│
├── paystackgin/
│   ├── go.mod
│   ├── middleware.go
│   ├── webhook.go
│   └── context.go
│
├── paystackfiber/
│   ├── go.mod
│   ├── middleware.go
│   ├── webhook.go
│   └── context.go
│
├── paystackecho/
│   ├── go.mod
│   ├── middleware.go
│   ├── webhook.go
│   └── context.go
│
└── testdata/
    ├── transaction_initialize.json
    ├── transaction_verify.json
    ├── transaction_list.json
    ├── transaction_list_empty.json
    ├── customer_create.json
    ├── customer_fetch.json
    ├── customer_list.json
    ├── plan_create.json
    ├── subscription_create.json
    ├── transfer_initiate.json
    ├── charge_create.json
    ├── refund_create.json
    ├── webhook_charge_success.json
    ├── error_401.json
    ├── error_400.json
    ├── error_400_fields.json
    ├── error_404.json
    ├── error_429.json
    ├── error_500.json
    └── error_html.html
```

---

## Philosophy

**Lean. Testable. Caller-autonomous. Extensible at every layer.**

The SDK represents the API faithfully. Business logic, retry strategies, currency
handling, and framework lifecycle belong to the caller or the integration package.
Every exported symbol is a maintenance burden — earn it.

---

## Stable Extension Points

These are the contracts framework integration packages depend on.
Never make a breaking change without a major semver bump.

### 1. `ClientInterface` (`client.go`)

The top-level contract. Framework packages accept and return this — never `*Client`.

```go
type ClientInterface interface {
    Transaction()  Transactor
    Customer()     Customeror
    Plan()         Planner
    Subscription() Subscriber
    Transfer()     Transferor
    Charge()       Charger
    Refund()       Refunder
}
```

`New(secretKey string, opts ...Option) (ClientInterface, error)`
- Validates key format (`sk_live_` or `sk_test_`) on construction. Fail fast.
- Returns `ClientInterface`. Never `*Client`. Callers never hold a concrete type.

### 2. `Backend` (`backend.go`)

The HTTP contract. Swap entirely for proxies, recording, framework test clients,
or in tests via `MockBackend`.

```go
type Backend interface {
    Call(ctx context.Context, method, path string, params, out interface{}) error
    CallRaw(ctx context.Context, method, path string, params interface{}) (*http.Response, error)
}

type BackendConfig struct {
    HTTPClient    *http.Client
    BaseURL       string
    Logger        Logger
    LeveledLogger LeveledLogger
}
```

`HTTPBackend` is the default implementation. It is exported so integration packages
can embed or wrap it.

### 3. Service Interfaces

Every service has an exported interface. Compile-time check in every service file.
Never remove it.

```
Transactor    → TransactionService
Customeror    → CustomerService
Planner       → PlanService
Subscriber    → SubscriptionService
Transferor    → TransferService
Charger       → ChargeService
Refunder      → RefundService
```

### 4. `Logger` and `LeveledLogger` (`logger.go`)

Integration packages pipe SDK logs into their framework's log system.
`Printf`-style and structured (Debugf/Infof/Warnf/Errorf) variants both supported.

### 5. `Event` and `EventType` (`webhook.go`)

Integration packages build webhook routers on top of these.
Never change the shape of `Event` without a major version bump.

`Event` has two fields: `Type EventType` and `Data json.RawMessage`.
Callers unmarshal `Data` into the concrete type they expect. No typed unions.

`EventType` constants cover all Paystack webhook events:
charge.success, transfer.success/failed/reversed, subscription.create/disable,
invoice.create/update, paymentrequest.pending/success.

---

## Core Implementation Rules

### `paystack.go`

- Package-level `var Key string` for simple usage, settable once at program start.
- Pointer helpers: `String`, `Int64`, `Bool`, `Float64`. Used for optional fields in
  request structs. Never force callers to take addresses of literals inline.

### `params.go`

- `Params` is the base for all request structs. Carries `IdempotencyKey` and `Metadata`.
  `IdempotencyKey` is sent as a header, not in the body.
- `ListParams` embeds `Params`. Adds `PerPage`, `Page`, `From`, `To` as pointer fields.
- `ListResponse[T any]` is the generic wrapper for all list endpoints.
- `Meta` is always exposed raw. Never hide pagination state from the caller.
- Empty `Data` with `Meta.Total > 0` is valid — never treat it as an error.
- Use `google/go-querystring` to encode `ListParams` into URL query params.

### `errors.go`

- One `*Error` type. Callers check `err.Code`. No type hierarchy.
- `ErrorCode` constants: `ErrCodeInvalidKey`, `ErrCodeInvalidRequest`,
  `ErrCodeNotFound`, `ErrCodeRateLimited`, `ErrCodeServerError`.
- `Fields map[string]string` populated on 400 validation errors.
- `RetryAfter time.Duration` parsed from `Retry-After` header on 429. Surface it. Never act on it.
- Network errors from `net/http` returned unwrapped.
- `Content-Type: text/html` responses return `ErrCodeServerError` with a descriptive
  message. Never attempt JSON decode on them.
- All errors work with `errors.As()`.

### `time.go`

- Custom `Time` wrapping `time.Time`.
- `UnmarshalJSON` tries all Paystack date layouts in order. Handled once here.
  Never handle date parsing in service files.

### `webhook.go`

- `Verify` uses `hmac.Equal` (constant-time). Never `==`.
- Body reads wrapped in `io.LimitReader`. Guard against oversized payloads.
- `ParseEvent` returns raw `Event`. Caller unmarshals `Data`.

### `backend.go` — `HTTPBackend`

- Single unexported `call()` drives all requests.
- Always drains and closes `resp.Body` even on error responses.
- Sets `Authorization: Bearer`, `Content-Type: application/json`, `User-Agent`.
- Passes `Idempotency-Key` header when `Params.IdempotencyKey` is set.
- Detects `Content-Type: text/html` before decoding. Returns `ErrCodeServerError`.

### Amounts

`int64` everywhere. Amounts are in **kobo** (1 NGN = 100 kobo).
Documented at package level and on every amount field. Never enforced by a custom type.

---

## Framework Integration Contract

All integration packages (`paystackgin`, `paystackfiber`, `paystackecho`):

- Are separate Go modules. Each has its own `go.mod`.
- Import `paystack-go` interfaces only. Never concrete structs.
- Accept `ClientInterface` in constructors. Never `*Client`.
- Expose identical signatures — only the framework context type differs:

```
Middleware(client ClientInterface) <framework.HandlerFunc>
ClientFrom(ctx) ClientInterface
WebhookHandler(secret []byte, fn func(*paystack.Event) error) <framework.HandlerFunc>
```

`WebhookHandler` verifies signature, enforces body size limit, parses the event,
calls `fn`. Returns appropriate HTTP status on error.

---

## Test Infrastructure (`internal/testutil/`)

Only imported in `_test.go` files. Never in production code.

### `MockBackend`

- Controls exactly what `Call()` returns via `Response interface{}` and `Err error`.
- Records every call in `Calls []CallRecord` for assertion.
- `LastCall()` returns the most recent record.
- Used when testing call behaviour: correct method, path, params sent.

### `FixtureBackend`

- Loads a JSON file from `testdata/` by name.
- Accepts `Status int` to simulate error responses.
- Accepts `Header http.Header` to simulate response headers (e.g. `Retry-After`).
- Used when the shape of the response matters, not just the call.

### `LoadFixture(t, name)`

Reads `testdata/<name>` relative to the test file. Calls `t.Fatal` if missing.
Fixture files are populated from real Paystack API responses — never hand-written.

---

## TDD Workflow

For every feature:
1. Write the interface method signature.
2. Write a table-driven test using `MockBackend` or `FixtureBackend`.
3. Run — must fail (red).
4. Implement the method.
5. Run — must pass (green).
6. Refactor. Run again.

---

## Test Rules

- All tests are table-driven. No single-case test functions.
- Every test function covers: happy path, all error codes, edge cases (empty list,
  missing fields, zero values, concurrent calls).
- Test unhappy paths first before writing the happy path.
- Never make real HTTP calls in unit tests. Use `MockBackend` or `FixtureBackend`.
- Integration package tests mock `ClientInterface` entirely — never instantiate `*Client`.
- Test service interfaces, not concrete structs. Tests must pass if the struct is swapped.
- Every `_test.go` file lives next to the file it tests. No separate `tests/` directory.
- Integration tests (real HTTP, real Paystack sandbox) live in `integration_test.go`
  with build tag `//go:build integration`. Never run in CI by default.

## Test Coverage Rules

- Floor: 80% overall.
- 100% required: `errors.go`, `webhook.go`, `time.go`.
- 100% required: all `internal/testutil/` files.
- Framework integration packages: 80% floor, 100% on `WebhookHandler`.

## What Every Service Test Must Cover

For every service method:
- Success response deserialises correctly.
- `401` returns `ErrCodeInvalidKey`.
- `400` returns `ErrCodeInvalidRequest` with `Fields` populated.
- `404` returns `ErrCodeNotFound`.
- `429` returns `ErrCodeRateLimited` with correct `RetryAfter`.
- `500` returns `ErrCodeServerError`.
- HTML response body returns `ErrCodeServerError` with descriptive message.
- Empty list (`data: []`) with `meta.total > 0` does not return an error.
- Context cancellation propagates correctly.
- Concurrent calls do not race (run with `-race`).

## What Webhook Tests Must Cover

- Valid signature verifies successfully.
- Tampered body fails verification.
- Tampered signature fails verification.
- Wrong secret fails verification.
- Oversized body is rejected before HMAC is computed.
- `ParseEvent` returns correct `Type` for every `EventType` constant.
- `ParseEvent` on malformed JSON returns an error.
- `WebhookHandler` (integration packages) returns 200 on success.
- `WebhookHandler` returns 400 on invalid signature.
- `WebhookHandler` returns 500 when handler `fn` returns an error.

---

## Versioning Rules

- **Stable** (`ClientInterface`, `Backend`, all service interfaces, `Event`,
  `Logger`, `LeveledLogger`): semver strict. Breaking change = major bump.
- **Internal** (concrete service structs, `HTTPBackend` internals): may change
  between minor versions. Safe because no external package holds them.
- **Request/response structs**: additive only. New fields in minor versions.
  Fields never removed or renamed without a major bump.

---

## Code Style

- `gofmt` and `golangci-lint` pass with zero warnings before every commit.
- No named return values.
- No `init()` functions.
- No global state except `paystack.Key`.
- Error strings: lowercase, no trailing punctuation.
- Prefer boring and readable over clever and compact.
- If a block needs a comment to be understood, extract it into a named function.

---

## What This SDK Does Not Do

- No retry logic. Return `*Error` with `RetryAfter`. Let the caller decide.
- No currency conversion. Kobo in, kobo out.
- No built-in logging. Callers pass `Logger` into `BackendConfig`.
- No `IsLive()`. The caller knows what key they passed in.
- No global default client beyond `paystack.Key`.
- No framework lifecycle management. That belongs in the integration packages.
