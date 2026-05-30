# Review: Phase 8.74 Runtime Overview Adapter Health

## Changes

- Added runtime overview dashboard proxy
  `/api/dashboard/runtime-overview/delivery-adapter-health`.
- Added a manual `Probe Health` action to the `Delivery Adapters` detail panel.
- The normal runtime overview read path still uses Go `GET /v1/runtime-overview`
  and does not call live adapter health automatically.
- Updated README, SPEC-014, SPEC-018, and the Chinese TODO.

## Review Notes

- The proxy is manual because Go adapter health can touch live OneBot/Telegram
  APIs through read-only `get_login_info` / `getMe`.
- The panel displays the returned JSON directly for operator readiness checks
  before QQ/NapCat live send smoke.
- No platform send API is called by this slice.

## Verification

- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-adapter-health`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py tests\test_agent_gateway_client.py tests\test_agent_gateway_outbox_worker.py -q --basetemp .tmp\pytest-runtime-overview-adapter-health-regression`
- `git diff --check`

## Decision

Accept as the browser-visible read-only preflight before live adapter smoke.
