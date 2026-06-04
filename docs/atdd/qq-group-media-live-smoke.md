# ATDD: QQ Group Media Live Smoke

## Scope

- Verify that the migrated local workspace can perform a real QQ group text send
  through Go `delivery-dispatch/send`.
- Verify that QQ group image/file sends are exercised against the real NapCat
  session and reported as either platform success or an explicit platform-side
  blocker.

## Preconditions

- `services/agent-runtime` is running on `127.0.0.1:8780`.
- Local NapCat container for `1049511700` is connected and healthy.
- `GET /v1/runtime-config` reports OneBot aliases configured.
- `GET /v1/delivery-adapters/health?timeout_seconds=5` reports
  `qq_1049511700` healthy and authenticated.
- `GET /v1/observe-targets` returns at least one enabled QQ group target for
  account `1049511700`.

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\run-qq-group-live-smoke.ps1`.
- Expect:
  The script emits JSON with one group text, one image, and one file case for a
  concrete group id. Every case must show `readiness.ready=true`.

### Scenario 2

- Action:
  Inspect the `text` case from the script output.
- Expect:
  `dispatch.results[*].status="sent"`, `provider_message_id` is non-empty, and
  the outbox record ends as `succeeded`.

### Scenario 3

- Action:
  Inspect the `image` and `file` cases from the script output.
- Expect:
  Each case either:
  - succeeds with a non-empty `provider_message_id` and final outbox status
    `succeeded`; or
  - fails with an explicit platform/runtime error string plus the resulting
    outbox state, without being reported as success.

## Failure Signals

- The script claims success while `dispatch` is null.
- `text` readiness is false or the text case lacks a provider message id.
- `image` or `file` failure is reduced to a generic “not run” state instead of a
  concrete blocker or platform error.
- The script reports Telegram as ready even though
  `runtime_config.delivery.telegram_token_configured=false`.

## Evidence

- Script JSON output.
- `GET /v1/runtime-config`.
- `GET /v1/delivery-adapters/health?timeout_seconds=5`.
- `GET /v1/observe-targets`.
- `GET /v1/outbox/{event_id}` for text/image/file cases.
