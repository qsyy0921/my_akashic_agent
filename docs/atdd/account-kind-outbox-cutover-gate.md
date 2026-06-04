# ATDD: Account-kind Outbox Cutover Gate

## Goal

Verify that Go local outbox execution can stay global-`text` while allowing
second-account QQ file delivery and still blocking first-account file delivery.

## Preconditions

- `agent-runtime` is healthy on `127.0.0.1:8780`
- OneBot dual-account QQ adapters are connected
- Python main process is running
- Go runtime is started with:
  - `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT=1049511700=text,2365524513=text|file`

## Acceptance Checks

1. Call `/v1/runtime-config` and confirm:
   - `workers.outbox_delivery_worker_enabled=true`
   - `workers.outbox_delivery_allowed_kinds=["text"]`
   - `workers.outbox_delivery_allowed_kinds_by_account.1049511700=["text"]`
   - `workers.outbox_delivery_allowed_kinds_by_account.2365524513=["text","file"]`
2. Call `/v1/queue-backend` and confirm:
   - `outbox_execution_owner=go_local_outbox_worker`
   - `outbox_execution_scope=account_kind_gated`
   - `outbox_allowed_kinds_by_account.2365524513=["text","file"]`
3. Call `/v1/runtime-workers` and confirm the outbox worker exposes
   `allowed_step_kinds_by_account=1049511700=text,2365524513=text|file`.
4. Post a real first-account group file outbox event and confirm repeated
   polling of `/v1/outbox/{event_id}` keeps:
   - `status=queued`
   - `attempts=0`
5. Post a real second-account group file outbox event and confirm it reaches:
   - `status=succeeded`
   - `attempts=1`
6. Call `/v1/agent-worker-statuses` and confirm Python outbox worker still
   reports `metadata.reason=go_runtime_outbox_worker_active`.
7. Call `/v1/outbound-cutover/readiness` and confirm:
   - `execution_ready=true`
   - `execution_owner=go_local_outbox_worker`
   - `attributes.outbox_execution_scope=account_kind_gated`

## Failure Signals

- first-account file is leased or failed instead of staying queued
- second-account file still requires manual `delivery-dispatch/send`
- runtime diagnostics omit the account-scoped gate
- Python outbox worker resumes leasing deliveries
