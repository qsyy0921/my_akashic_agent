# Review: receiver status diagnostics

## Scope

- Added Go-owned receiver lifecycle diagnostics for Python platform receivers.
- Added `POST /v1/receiver-statuses/report` and `GET /v1/receiver-statuses`.
- Added runtime overview summary fields and a `Receiver Statuses` card.
- Wired Python QQ startup success/failure and Telegram polling/conflict state
  to report into Go.

## Boundaries

- Go stores and aggregates the latest receiver status only.
- Python still owns NcatBot and Telegram SDK receive loops.
- This slice is read-only from the operator perspective and does not start,
  stop, reconnect, or send platform messages.

## Validation

- `go test ./...`
- `go vet ./...`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-receiver-status`
- `uv run python -m py_compile bootstrap\channels.py infra\channels\telegram_channel.py integrations\agent_gateway.py plugins\runtime_overview\dashboard.py`

## Live Smoke

- Rebuilt and restarted local `agent-runtime` and Python dashboard.
- `GET /v1/receiver-statuses` returned three connected receivers:
  `qq:1049511700:qq`, `qq:2365524513:qq_2365524513`, and
  `telegram:7689386159:telegram`.
- Runtime overview reported `receiver_statuses=3`,
  `receiver_status_connected=3`, `receiver_status_qq=2`, and
  `receiver_status_telegram=1`.
- QQ observe-only inbox metrics continued to record live group messages.

## Follow-up

- Ask the operator to send one image and one file in an observe-only QQ group
  to verify attachment capture end to end.
- If Telegram `getUpdates` conflict returns, the new receiver status endpoint
  should show `status=suspended` with `reason=getupdates_conflict`; then locate
  and stop the duplicate polling instance or move to webhook/single-instance
  locking.
