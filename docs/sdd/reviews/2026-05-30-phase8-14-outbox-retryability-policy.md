# Review: Outbox Retryability Policy

Spec:

- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`

Implementation summary:

- Added Go domain retryability rules to `DeliveryErrorKind`.
- `route_error`, `unsupported_media`, and `validation_error` now move an outbox
  delivery directly to `dead_lettered` because retrying them cannot succeed
  without changing route, payload, or configuration.
- `unknown`, `platform_error`, `platform_timeout`, and `sender_unavailable`
  remain retryable until max attempts are exhausted.
- Added domain, app-service, and HTTP tests covering retryable and
  non-retryable failure behavior.

Tests run:

- Go `go test ./...` under `services/agent-runtime`.
- Go `go build -o ..\..\.tmp\bin\agent-runtime-retryability-smoke.exe .\cmd\agent-runtime`.

Findings:

- This keeps retry/dead-letter decisions in the Go domain instead of Python
  worker code or dashboard operations.
- Existing callers that omit `error_kind` still get `unknown`, preserving the
  historical retryable behavior.

Decision:

- Accept as a Go-owned delivery policy slice required before real platform
  adapter cutover.

Follow-ups:

- Add dashboard aggregate counts for retryable vs non-retryable failures.
- Revisit `sender_unavailable` once real Go platform adapters can distinguish
  temporary service outage from missing configuration.
