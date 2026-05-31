# Review: receiver status heartbeat

## Scope

- Added Go file-backed receiver status repository selected by
  `AKASHIC_RECEIVER_STATUSES_DSN`, `AKASHIC_RECEIVER_STATUSES_PATH`, or the
  default runtime state directory.
- Added read-time stale heartbeat handling with
  `AKASHIC_RECEIVER_STATUS_STALE_SECONDS`.
- Wired `agent-runtime` startup to load persisted receiver statuses.
- Added Python QQ/NapCat receiver heartbeat and shutdown `stopped` reporting.
- Added Telegram receiver status heartbeat independent of receiver lease renewal,
  while preserving `getupdates_conflict` suspension semantics.
- Documented the SDD decision and updated the Chinese migration TODO.

## Boundaries

- No QQ/Telegram platform sends are triggered by this slice.
- Python still owns platform receive loops and SDK callbacks.
- Go owns durable status state and read-only diagnostics only.
- Receiver leases remain in-memory in this slice; status recovery is intentionally
  separate from lease ownership.

## Validation

- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests/test_bootstrap_wiring_p2.py::test_config_load_reads_agent_gateway_integration_block tests/test_bootstrap_wiring_p2.py::test_config_load_reads_agent_runtime_integration_block_with_compatibility tests/test_channel_clients.py::test_telegram_channel_acquires_receiver_lease tests/test_channel_clients.py::test_telegram_channel_skips_polling_when_receiver_lease_is_held -q --basetemp .tmp\pytest-receiver-heartbeat`
- `uv run python -m py_compile bootstrap\channels.py infra\channels\telegram_channel.py integrations\agent_gateway.py agent\config.py agent\config_models.py`
- `git diff --check`

## Runtime Smoke

- Rebuilt and restarted local `agent-runtime` on `:8780` with the current
  OneBot WebSocket aliases and Telegram adapter environment; health,
  `/v1/runtime-config`, `/v1/receiver-statuses`, and `/v1/runtime-overview`
  responded.
- Started an isolated smoke runtime on `:18780` with
  `AKASHIC_RUNTIME_STATE_DIR=.tmp\receiver-status-smoke-state`, posted a
  synthetic receiver status, stopped and restarted that runtime, then verified
  `GET /v1/receiver-statuses` still returned the receiver from
  `receiver-statuses.json`.
- No QQ/Telegram platform sends were executed. The running Python receiver
  process was not restarted in this smoke, so live QQ/Telegram heartbeats will
  appear after the next Python receiver startup or reload.

## Follow-up

- If Telegram polling conflict handling still needs stricter single-owner
  guarantees after Go restart, add durable receiver lease recovery/reacquire
  semantics as a separate cutover-gated slice.
