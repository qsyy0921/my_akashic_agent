# Review: Phase 8.61 Runtime Queue Dashboard

## Scope

- Added `AgentGatewayClient.get_queue_backend()` for the Go-owned
  `GET /v1/queue-backend` diagnostic endpoint.
- Extended runtime overview dashboard aggregation with queue backend provider,
  mode, consumer concurrency, max-in-flight, and external lease gate status.
- Kept the path read-only; it does not start NATS consumers, acknowledge queue
  messages, or execute platform/Python side effects.

## Design

- Go remains the source of truth for queue ownership and cutover gates.
- Python dashboard only normalizes the Go view for operator visibility.
- The dashboard highlights blocked `external_lease` gate state as `warn`, while
  local state-store mode remains `ok`.
- This keeps the multi-goroutine MQ design visible before enabling any
  cutover flags.

## Verification

- `uv run pytest tests\test_agent_gateway_client.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-queue`

## Remaining Risk

- This is diagnostic visibility only. It proves the UI can display Go-owned MQ
  state, but it does not prove a live NATS broker is running. Live NATS smoke
  remains the separate `AKASHIC_NATS_SMOKE_DSN` gated test path.
