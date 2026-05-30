# Review: Phase 8.76 Runtime Config Live Preflight

## Changes

- Hardened `GET /v1/runtime-config` secret redaction so token/secret
  environment variables return only `redacted` or `channel=redacted`, not
  partial token prefixes/suffixes.
- Kept `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN` visible as a boolean runtime flag
  rather than treating it as a secret value.
- Rebuilt and restarted the local Go `agent-runtime` binary on `:8780`.
- Restarted the Python `main.py` process so the dashboard mounts the current
  `runtime_overview` plugin routes.
- Updated SPEC-014, SPEC-018, and the Chinese TODO.

## Live Verification

- `GET /v1/runtime-config`: returned 200; OneBot dual-account readiness was true,
  missing alias list was empty, token preview was `redacted`, strict token flag
  was `false`.
- `GET /v1/delivery-adapters`: returned OneBot aliases `qq`, `qq_1049511700`,
  `qq_2365524513`, plus Telegram, all enabled.
- `GET /v1/delivery-adapters/health?timeout_seconds=5`: returned healthy and
  authenticated for `1049511700`, `2365524513`, and Telegram using read-only
  probes.
- `GET /v1/runtime-overview`: returned `Delivery Adapters=4` and
  `Runtime Config=:8780` with status `ok`.
- `GET /api/dashboard/runtime-overview`: returned 200 after Python restart.
- `GET /api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=5`:
  returned 200 through the dashboard proxy.

## Safety Notes

- Go `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=false`; no Go-owned automatic
  delivery worker is running.
- The health checks use OneBot `get_login_info` and Telegram `getMe`; no QQ or
  Telegram send API was called.
- QQ/NapCat live send smoke remains a separate explicit cutover task.

## Verification

- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-live-preflight`
- `git diff --check`

## Decision

Accept as the live read-only readiness gate before any QQ/NapCat Go send smoke.
