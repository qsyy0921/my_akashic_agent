# Review: Phase 8.60 Runtime Adapter Dashboard

## Scope

- Added `AgentGatewayClient.list_delivery_adapters()` for the Go-owned
  `GET /v1/delivery-adapters` diagnostic endpoint.
- Extended runtime overview dashboard aggregation to read delivery adapter
  diagnostics and summarize enabled/disabled adapters.
- Kept the path strictly read-only; it does not call Telegram or OneBot send
  APIs.

## Design

- Go remains the source of truth for adapter configuration diagnostics.
- Python client and dashboard only consume the Go endpoint and normalize fields
  for operator visibility.
- The dashboard surfaces adapter readiness next to job leases, dead letters,
  checkpoint lag, and event stream summaries, so QQ/Telegram cutover checks do
  not depend on reading logs.

## Verification

- `uv run pytest tests\test_agent_gateway_client.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-adapters`

## Remaining Risk

- This is still configuration visibility, not a platform health check. A future
  explicit live smoke can call read-only OneBot `get_login_info`, but real send
  smoke should remain manual and operator-approved.
