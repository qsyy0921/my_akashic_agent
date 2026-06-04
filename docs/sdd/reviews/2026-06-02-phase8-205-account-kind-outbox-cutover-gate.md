# Review: account-kind outbox cutover gate

## Scope

- `docs/sdd/specs/agent-gateway/148-account-kind-outbox-cutover-gate.md`
- `services/agent-runtime/app/command/outbox.go`
- `services/agent-runtime/app/port/out/outbox.go`
- `services/agent-runtime/app/query/queue_backend.go`
- `services/agent-runtime/app/query/runtime_config.go`
- `services/agent-runtime/app/service/outbox_service.go`
- `services/agent-runtime/cmd/agent-runtime/*`
- `services/agent-runtime/trigger/job/outbox_delivery_worker.go`
- `integrations/agent_gateway_outbox_worker.py`

## What changed

- Added `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT` parsing and runtime
  diagnostics.
- Extended outbox lease filtering so account-scoped allowed kinds override the
  global gate for matching account ids.
- Kept Python compatibility outbox worker in backoff while Go local outbox
  execution is active.
- Added tests for service-level lease filtering, persistent store lease
  filtering, worker propagation, and runtime-config exposure.

## Verification

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...`
- `uv run pytest tests/test_agent_gateway_outbox_worker.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- Re-started runtime with:
  - `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT=1049511700=text,2365524513=text|file`
- Verified live runtime state:
  - `/v1/runtime-config`
  - `/v1/queue-backend`
  - `/v1/runtime-workers`
  - `POST /v1/outbound-cutover/readiness`
- Posted real outbox smoke:
  - first-account file `qq-account-gate:first-file:5ada75b397494cc6bfd61ed50d707084`
    stayed `queued`
  - second-account file
    `qq-account-gate:second-file:8ffb2b622ed1420f833447943e8720f0`
    reached `succeeded`

## Result

- Accept. Go outbox ownership is now more precise than global `text_only`:
  second-account file delivery can run on Go local worker without globally
  exposing first-account rich-media.
- Goal remains active because Telegram backend still lacks token and QQ image
  still fails natively across both accounts.
