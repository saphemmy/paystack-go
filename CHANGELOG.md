# Changelog

All notable changes are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] — 2026-04-16

### Added

- Initial public release.
- `ClientInterface` and `New(secretKey, opts...)` with key-format validation.
- `Backend` interface and the default `HTTPBackend` implementation.
- Seven resource services behind stable interfaces:
  `Transactor`, `Customeror`, `Planner`, `Subscriber`,
  `Transferor`, `Charger`, `Refunder`.
- Webhook primitives: `Verify`, `ParseEvent`, `ParseWebhook`, the full
  `EventType` constant set, and `ErrInvalidSignature`.
- Framework adapters (each in its own Go module):
  `paystackgin`, `paystackfiber`, `paystackecho`, each exposing
  `Middleware`, `ClientFrom`, and `WebhookHandler`.
- Unified `*Error` with `Code`, `Fields`, `RetryAfter`, and `StatusCode`;
  works with `errors.As`.
- `Logger` and `LeveledLogger` hooks.
- Amounts standardised to int64 kobo end to end.

[Unreleased]: https://github.com/saphemmy/paystack-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/saphemmy/paystack-go/releases/tag/v0.1.0
