# ATDD: Local Outbox Worker Private Text Smoke

## Scope

- Verify that `go_local_outbox_worker` can become the active outbox execution
  owner on the local migrated workspace.
- Verify that a real QQ private-text outbox item can be sent automatically by
  the worker without manual `delivery-dispatch/send`.

## Preconditions

- `services/agent-runtime` can be restarted locally.
- Local NapCat containers for `1049511700` and `2365524513` are logged in.
- `GET /v1/delivery-adapters/health?timeout_seconds=5` reports the QQ aliases
  healthy and authenticated.

## Scenarios

### Scenario 1

- Action:
  Start `agent-runtime` with `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` and a
  valid `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` mapping.
- Expect:
  `GET /v1/runtime-workers` shows `outbox_delivery_worker.enabled=true` and
  `running=true`, and `POST /v1/outbound-cutover/readiness` shows
  `execution_ready=true` with `execution_owner=go_local_outbox_worker`.

### Scenario 2

- Action:
  Create one QQ private-text outbox event for
  `1049511700 -> 2365524513` using `POST /v1/outbound`, then wait for the
  worker to process it.
- Expect:
  The outbox record reaches `status=succeeded` without a manual
  `delivery-dispatch/send` call, `attempts=1`, and NapCat logs show the private
  send.

### Scenario 3

- Action:
  Roll the runtime back to the default local bring-up script.
- Expect:
  `GET /v1/runtime-config` returns
  `outbox_delivery_worker_enabled=false`, and
  `POST /v1/outbound-cutover/readiness` returns
  `execution_ready=false` with `outbox_execution_path_not_ready`.

## Failure Signals

- Worker is enabled but `runtime-workers` still shows it not running.
- Automatic outbox send reaches `dead_lettered` or `failed` after the platform
  send actually succeeded.
- Rollback does not restore the default worker-disabled state.

## Evidence

- `GET /v1/runtime-workers`
- `POST /v1/outbound-cutover/readiness`
- `GET /v1/outbox/{event_id}`
- `GET /v1/send-ledger/recent?...`
- NapCat logs showing the private send
