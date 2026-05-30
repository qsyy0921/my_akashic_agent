# Review: Phase 8.72 Runtime Worker Diagnostics

## Changes

- Added Go-owned `GET /v1/runtime-workers` for read-only worker diagnostics.
- The endpoint reports enabled/running state for `agent_job_recovery`,
  `outbox_delivery_worker`, `nats_shadow_publisher`, `nats_dual_read_compare`,
  and `nats_external_lease`.
- Extended `GET /v1/runtime-overview` and the Python runtime overview dashboard
  with runtime worker totals and a `runtime_workers` card.
- Documented the endpoint in README, SPEC-014, and the Chinese migration TODO.

## Review Notes

- This slice is diagnostic only. It does not start workers, send QQ/Telegram
  messages, acknowledge NATS deliveries, or mutate state.
- Runtime worker data is derived from startup environment/config plus the
  existing queue backend view. It should be treated as a readiness snapshot, not
  a replacement for logs when a goroutine exits after startup.
- NATS external lease remains gated by existing cutover and smoke flags.

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-worker-diagnostics`

## Decision

Accept as the read-only worker control-plane view needed before live OneBot and
NATS cutover smoke.
