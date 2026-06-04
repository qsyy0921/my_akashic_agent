# ATDD: Text-only Outbox Cutover Gate

## Goal

Verify that QQ text delivery can be owned by Go while rich-media delivery stays
gated off in the outbox until the NapCat / QQ platform blocker is removed.

## Preconditions

- `agent-runtime` is healthy on `127.0.0.1:8780`
- OneBot dual-account QQ adapters are connected
- Go runtime is started with:
  - `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
- Python main process is running

## Acceptance Checks

1. Call `/v1/runtime-config` and confirm:
   - `workers.outbox_delivery_worker_enabled=true`
   - `workers.outbox_delivery_allowed_kinds=["text"]`
   - `delivery.telegram_token_configured=false` if token is still absent
2. Call `/v1/queue-backend` and confirm:
   - `outbox_execution_owner=go_local_outbox_worker`
   - `outbox_execution_scope=text_only`
   - `outbox_allowed_kinds=["text"]`
3. Call `/v1/outbound-cutover/readiness` and confirm:
   - `execution_ready=true`
   - `execution_owner=go_local_outbox_worker`
4. Post a real QQ private text outbox event through `/v1/outbound` and confirm
   it reaches `succeeded` without manual `delivery-dispatch/send` or manual
   `POST /v1/outbox/{event_id}/succeeded`.
5. Inspect `/v1/outbox-events?limit=20` and confirm the text event contains:
   - `queued`
   - `leased` with `lease_owner=agent-runtime-outbox-worker`
   - `succeeded`
6. Post a real QQ group image outbox event and confirm repeated polling of
   `/v1/outbox/{event_id}` keeps:
   - `status=queued`
   - `attempts=0`
7. Post a real QQ group file outbox event and confirm repeated polling of
   `/v1/outbox/{event_id}` keeps:
   - `status=queued`
   - `attempts=0`
8. Call `/v1/agent-worker-statuses` and confirm Python outbox worker reports
   `metadata.reason=go_runtime_outbox_worker_active`.
9. Call `/v1/knowledge-job-planner/readiness` and `/v1/jobs?type=group_memory_extract`
   and confirm Go planner remains active and new jobs still carry
   `metadata.scheduler=agent-runtime-knowledge-job-planner`.

## Failure Signals

- text delivery still needs manual dispatch or manual mark-succeeded
- image/file deliveries are leased or failed instead of staying queued
- Python outbox worker still becomes the execution owner
- `execution_ready` falls back to false
