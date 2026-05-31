# Review: receiver lease control

## Scope

- Added Go-owned receiver lease control for long-running Python platform
  receivers.
- Added `POST /v1/receiver-leases/acquire`,
  `POST /v1/receiver-leases/renew`, `POST /v1/receiver-leases/release`, and
  `GET /v1/receiver-leases`.
- Added runtime overview receiver lease summary fields and card.
- Wired Telegram polling to acquire a Go lease before `start_polling`, renew
  it in the background, and release it on stop/conflict.

## Boundaries

- Go owns the single-instance runtime decision, lease token, TTL, active/expired
  state, and dashboard counters.
- Python still owns Telegram SDK setup, polling, send helpers, and conflict
  handling.
- QQ receiver leases remain diagnostic-only for now; the slice only gates
  Telegram polling to avoid destabilizing working NapCat observation.

## Validation

- `go test ./...`
- `go vet ./...`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_runtime_overview_dashboard_plugin.py tests\test_channel_clients.py::test_telegram_channel_acquires_receiver_lease tests\test_channel_clients.py::test_telegram_channel_skips_polling_when_receiver_lease_is_held -q --basetemp .tmp\pytest-receiver-lease-final`
- `uv run python -m py_compile infra\channels\telegram_channel.py integrations\agent_gateway.py plugins\runtime_overview\dashboard.py`

## Runtime Smoke

- Rebuild and restart local `agent-runtime` plus Python dashboard.
- Verify `GET /v1/receiver-leases` shows one active Telegram lease for
  `telegram:7689386159:telegram`, with `lease_token_present=true` and no raw
  token in list output.
- Verify `GET /v1/receiver-statuses` shows two QQ receivers and one Telegram
  receiver connected.
- Verify Go `GET /v1/runtime-overview` includes `receiver_leases=1`,
  `receiver_leases_active=1`, and `receiver_leases_expired=0`.
- Confirm dashboard `/api/dashboard/runtime-overview` normalizes the same
  receiver lease summary.

## Follow-up

- If an external non-Akashic Telegram polling process is still running, Go
  lease control will not stop it; receiver status should still show
  `getupdates_conflict`.
- Later slices can add QQ receiver leases after image/file observe-only capture
  is verified end to end.
