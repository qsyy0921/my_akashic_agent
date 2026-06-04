# ATDD: QQ Live Send Smoke

## Scope

- Verify that the migrated local workspace can perform a real QQ/NapCat
  private-text send through Go `delivery-dispatch/send` without enabling the
  full Go local outbox worker.

## Preconditions

- `services/agent-runtime` is running on `127.0.0.1:8780`.
- Local NapCat containers are running and authenticated for:
  - `1049511700`
  - `2365524513`
- `GET /v1/runtime-config` reports OneBot aliases configured.
- `GET /v1/delivery-adapters/health?timeout_seconds=5` reports the QQ aliases
  healthy and authenticated.

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\run-qq-live-smoke.ps1`.
- Expect:
  The script emits JSON showing both private directions, each with:
  - `readiness.ready=true`
  - `dispatch.results[*].status="sent"`
  - non-empty `provider_message_id`
  - final outbox status `succeeded` when `SkipMarkSucceeded` is not used

### Scenario 2

- Action:
  Run the script while OneBot adapter health is not ready or the target channel
  alias is missing.
- Expect:
  The script fails before claiming success and surfaces the readiness reason or
  adapter error; it does not report Telegram as ready and does not modify any
  non-smoke outbox event.

## Failure Signals

- `delivery-dispatch/readiness` returns `ready=false`.
- `delivery-dispatch/send` returns adapter unavailable or platform error.
- Outbox state is left queued or dispatching after a reported success.
- The script claims Telegram success even though
  `runtime_config.delivery.telegram_token_configured=false`.

## Evidence

- Script JSON output.
- `GET /v1/runtime-config`.
- `GET /v1/delivery-adapters/health?timeout_seconds=5`.
- `GET /v1/outbox/{event_id}` for both smoke event ids.
