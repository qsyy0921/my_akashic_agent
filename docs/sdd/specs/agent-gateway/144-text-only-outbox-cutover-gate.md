# 144 Text-only Outbox Cutover Gate

## Context

The Go runtime already supports `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
and can lease/dispatch QQ private text through `go_local_outbox_worker`.
But the Python compatibility outbox worker could still lease rich-media
deliveries from the same state store, which broke the intended partial cutover:
text should move to Go, while image/file deliveries should stay queued until the
NapCat / QQ rich-media platform blocker is resolved.

## Decision

1. When Go runtime reports `workers.outbox_delivery_worker_enabled=true`, the
   Python compatibility outbox worker must back off before leasing any outbox
   delivery.
2. Repo-local runtime bring-up must accept an explicit
   `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS` value so text-only cutover is
   reproducible from the repo launcher.
3. Text-only cutover is considered valid only if:
   - a real text outbox event is automatically leased and completed by
     `agent-runtime-outbox-worker`;
   - real image/file outbox events remain `queued` with `attempts=0`.

## Boundary

### Go owns

- outbox execution ownership when `go_local_outbox_worker` is enabled
- allowed step kind filtering for the local outbox worker
- text-only lease/dispatch behavior and outbox lifecycle transitions

### Python owns

- compatibility outbox execution only when Go local outbox ownership is not
  active
- AI runtime, message/tool orchestration, knowledge/image execution

## Non-goals

- No attempt to make QQ rich media succeed through Akashic while NapCat /
  QQ native parity still fails
- No migration of Telegram bot backend without a token
- No NATS external-lease cutover in this slice

## Validation

- `/v1/runtime-config` reports:
  - `outbox_delivery_worker_enabled=true`
  - `outbox_delivery_allowed_kinds=["text"]`
- `/v1/queue-backend` reports:
  - `outbox_execution_owner=go_local_outbox_worker`
  - `outbox_execution_scope=text_only`
- `/v1/outbound-cutover/readiness` reports:
  - `execution_ready=true`
  - `execution_owner=go_local_outbox_worker`
  - `attributes.outbox_allowed_kinds=text`
- A real QQ private text outbox event transitions
  `queued -> leased -> succeeded` with `lease_owner=agent-runtime-outbox-worker`
- Real QQ group image/file outbox events remain `queued` over repeated polls and
  never increment `attempts`
- `/v1/agent-worker-statuses` shows Python outbox worker idle with
  `reason=go_runtime_outbox_worker_active`
