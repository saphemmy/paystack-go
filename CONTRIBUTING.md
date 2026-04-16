# Contributing

Thanks for your interest in improving `paystack-go`. This document covers the
ground rules. The design philosophy and implementation constraints are
documented in [ARCHITECTURE.md](./ARCHITECTURE.md) — please read it before opening a
substantial PR.

## Ground rules

- **Be kind.** We follow the [Contributor Covenant](./CODE_OF_CONDUCT.md).
- **Open an issue first** for anything larger than a bug fix or docs tweak.
  Design discussion is much cheaper than wasted implementation.
- **One logical change per PR.** Don't mix refactors with features.
- **No new dependencies** without discussion. The SDK keeps a small dep
  surface on purpose.

## Development setup

Requirements:

- Go 1.22 or later
- `golangci-lint` ([install](https://golangci-lint.run/usage/install/))

```bash
git clone git@github.com:saphemmy/paystack-go.git
cd paystack-go
go test ./...
```

## Workflow

1. Fork the repo and create a topic branch off `main`.
2. Follow the TDD workflow described in [ARCHITECTURE.md](./ARCHITECTURE.md):
   - Write the interface method signature.
   - Write a failing table-driven test using `MockBackend` or `FixtureBackend`.
   - Implement the method.
   - Run `go test -race ./...`.
3. Run the full checklist before opening a PR:

   ```bash
   gofmt -s -w .
   go vet ./...
   golangci-lint run
   go test -race -cover ./...
   ```

4. Open a PR with a clear description of the change and its motivation. Link
   the issue it resolves.

## Commit messages

Short, imperative subject line (≤ 72 chars). Body explains the *why*, not the
*what* — the diff already shows what changed. Examples:

```
Add idempotency-key header to HTTPBackend

The Paystack API accepts Idempotency-Key on every write endpoint. Surfacing
it through params.Params lets callers opt in without a backend swap.
```

## Testing rules

Summarised from [ARCHITECTURE.md](./ARCHITECTURE.md):

- All tests are table-driven. No single-case test functions.
- Unhappy paths first, then the happy path.
- Never make real HTTP calls in unit tests. Use `MockBackend` or
  `FixtureBackend`.
- Coverage floor is 80% overall; 100% on `errors.go`, `webhook.go`, `time.go`,
  and everything under `internal/testutil/`.
- Integration tests (real sandbox) live behind the `integration` build tag.

## Releasing

Releases follow semver and are tagged `vX.Y.Z`. The stable surface —
`ClientInterface`, `Backend`, service interfaces, `Event`, `Logger`,
`LeveledLogger` — never changes without a major bump. Request/response structs
may only grow additively in minor releases.
