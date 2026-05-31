# Review: receiver lease persistence

## Scope

- Added Go file-backed receiver lease repository selected by
  `AKASHIC_RECEIVER_LEASES_DSN`, `AKASHIC_RECEIVER_LEASES_PATH`, or the default
  runtime state directory.
- Wired `agent-runtime` startup to load persisted receiver leases together with
  receiver statuses.
- Persisted receiver lease acquire, renew, and release mutations.
- Added explicit expired-lease rejection during renew, while acquire can still
  replace expired leases.
- Added Telegram renew recovery: missing/expired/token-mismatch lease errors
  attempt reacquire; if another holder owns the lease, polling is suspended.
- Updated SDD, README, runtime config diagnostics, and the Chinese migration
  TODO.

## Boundaries

- No QQ/Telegram platform sends are triggered by this slice.
- Python still owns Telegram SDK polling and only uses Go for lease control.
- QQ receiver leases remain out of scope until observe-only media capture is
  verified as stable.

## Validation

- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests/test_channel_clients.py::test_telegram_channel_acquires_receiver_lease tests/test_channel_clients.py::test_telegram_channel_skips_polling_when_receiver_lease_is_held tests/test_channel_clients.py::test_telegram_channel_reacquires_receiver_lease_after_runtime_restart tests/test_agent_gateway_client.py::test_agent_gateway_client_reports_and_lists_receiver_statuses tests/test_agent_gateway_client.py::test_agent_gateway_client_manages_receiver_leases -q --basetemp .tmp\pytest-receiver-lease`
- `uv run python -m py_compile infra\channels\telegram_channel.py integrations\agent_gateway.py agent\config.py agent\config_models.py`
- `git diff --check`

## Runtime Smoke

- Started an isolated `agent-runtime` on `:18781` with
  `AKASHIC_RUNTIME_STATE_DIR=.tmp\receiver-lease-smoke-state`.
- Acquired a synthetic Telegram receiver lease for
  `telegram:codex-smoke:telegram`, stopped the runtime, restarted it, and
  renewed the same lease token successfully.
- Verified `receiver-leases.json` exists and `GET /v1/receiver-leases` returns
  one lease after restart.
- Rebuilt and restarted the current local `agent-runtime` on `:8780`; health
  returned OK and `/v1/runtime-config` exposed the new receiver lease override
  environment keys.
- No QQ/Telegram platform sends were executed.
