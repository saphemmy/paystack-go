# Security Policy

`paystack-go` handles financial flows. We treat security reports seriously
and respond quickly.

## Supported versions

The SDK is pre-1.0. Security fixes land on the latest `v0.x` minor and are
tagged as patch releases.

| Version | Supported          |
| ------- | ------------------ |
| `v0.1.x`| ✅ yes             |
| `< v0.1`| ❌ no              |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security problems.**

Report privately through either channel:

1. **Preferred — GitHub Security Advisories**
   <https://github.com/saphemmy/paystack-go/security/advisories/new>
   (private to maintainers; includes a CVE request workflow).

2. **Email** — `phemmy0889@gmail.com` with subject `[paystack-go security]`.

Include, where possible:

- Affected version or commit SHA
- Reproduction steps or a proof-of-concept
- The observed behaviour and the expected behaviour
- Any mitigation you're already aware of

## What to expect

- **Acknowledgement** within 72 hours of receipt.
- **Triage + severity assessment** within 7 days, using CVSS 3.1.
- **Fix + advisory + patched release** on an agreed timeline (typical
  target: 30 days for high severity, faster for criticals; longer for
  low-impact issues with working mitigations).
- **Credit** in the advisory unless you request anonymity.

## In scope

- Signature-verification bypass in `Verify` / `ParseWebhook`.
- HMAC timing or truncation attacks.
- Leakage of secret keys through logs or panics.
- Unexpected mutation of caller state.
- Any issue that could cause the SDK to accept a forged Paystack response.

## Out of scope

- Vulnerabilities in the upstream Paystack API — please report those to
  Paystack directly (<security@paystack.com>).
- Denial-of-service from a caller's own infrastructure (HTTP client
  timeouts, memory pressure under caller-controlled workloads).
- Issues in third-party dependencies that do not affect the SDK's
  behaviour; those should be reported upstream.

Thanks for helping keep the ecosystem safe.
