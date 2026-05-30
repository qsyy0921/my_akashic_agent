# Review: Outbox Structured Failure Kind

Spec:

- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`

Implementation summary:

- Added Go domain-level `DeliveryErrorKind` values for outbox delivery failures:
  `unknown`, `platform_error`, `platform_timeout`, `route_error`,
  `unsupported_media`, `sender_unavailable`, and `validation_error`.
- Extended `OutboxDelivery`, app commands, query views, and HTTP state updates
  with optional `error_kind` while preserving existing `error_message`
  behavior.
- Updated the Python agent-runtime outbox compatibility worker to classify
  common `message_push` failure results and report structured failure kinds to
  Go.
- Exposed `error_kind` in the dashboard outbox adapter and panel so delivery
  failures can be filtered and inspected before platform adapter cutover.

Tests run:

- Go `go test ./...` under `services/agent-runtime`.
- Python pytest for agent runtime client, outbox worker, and outbox dashboard
  plugin.
- Python compile check for changed Python modules.
- Node syntax check for the outbox dashboard panel.

Findings:

- This is still a non-production control-plane slice. Actual QQ/Telegram SDK
  sends remain in the Python compatibility layer.
- Failure kind is normalized in Go domain logic; omitted or unknown values
  become `unknown` instead of leaking arbitrary strings into durable state.
- `retry`, `dispatching`, and `succeeded` clear previous failure details so
  stale adapter errors do not remain on active deliveries.

Decision:

- Accept as a Go-owned delivery lifecycle improvement required before adapter
  cutover and alerting.

Follow-ups:

- Add aggregate dashboard counts by `error_kind` after enough runtime samples
  exist.
- Map real Go platform adapter errors to the same `DeliveryErrorKind` values
  when QQ/Telegram delivery adapters are reviewed for cutover.
