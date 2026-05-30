# Review: Phase 8.63 Outbox Worker Readiness Gate

## Scope

- Updated the Python compatibility outbox worker to call Go
  `POST /v1/delivery-dispatch/readiness` before attempting Go runtime dispatch
  for configured runtime-dispatch channels.
- If Go reports `ready=true`, the worker proceeds with Go runtime dispatch.
- If Go reports `ready=false`, the worker executes the Go-provided plan through
  the Python sender fallback and does not call Go dispatch.
- Go readiness planning errors are still written back with Go-provided
  `error_kind`.

## Design

- Go owns deterministic route planning and adapter readiness.
- Python remains the compatibility sender and fallback executor.
- The worker keeps the previous behavior when readiness is unavailable, so old
  runtimes remain compatible during rollout.
- This avoids unnecessary failed Go dispatch attempts when QQ channel aliases
  are listed in `outbound_channels` before a matching DeliveryAdapter is active.

## Verification

- `uv run pytest tests\test_agent_gateway_outbox_worker.py -q --basetemp .tmp\pytest-outbox-readiness-worker`

## Remaining Risk

- Readiness checks still reflect configuration, not live platform health.
  Real QQ/Telegram send smoke remains explicit and operator-approved.
