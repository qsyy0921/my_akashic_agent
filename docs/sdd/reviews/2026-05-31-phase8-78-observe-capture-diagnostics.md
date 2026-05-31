# Review: observe capture diagnostics

## Scope

- Added Go-owned `GET /v1/observe-capture-diagnostics`.
- Joined observe targets, receiver statuses, inbox events, media assets, and
  safe local media content probes into one read-only coverage view.
- Added runtime overview summary fields and an `Observe Capture` card.
- Added Python `AgentGatewayClient.get_observe_capture_diagnostics` and
  dashboard normalization for the Go aggregate.

## Boundaries

- Go owns deterministic diagnostics: target coverage, receiver connectivity,
  media registry counts, and safe local content readiness.
- Python still owns QQ/NapCat callbacks, media download, group file URL lookup,
  and OCR/VLM/model summaries.
- This slice does not change observe-only reply behavior and does not trigger
  real QQ/Telegram sends.

## Validation

- `go test ./...`
- `go vet ./...`
- `uv run pytest tests\test_agent_gateway_client.py::test_agent_gateway_client_gets_observe_capture_diagnostics tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-observe-capture-final`
- `uv run python -m py_compile integrations\agent_gateway.py plugins\runtime_overview\dashboard.py`

## Runtime Smoke

- Rebuild and restart local `agent-runtime` plus Python dashboard.
- Verify `GET /v1/observe-capture-diagnostics` returns six configured QQ
  observe-only targets with `side_effect=none`.
- Current live result: all six targets have `receiver_connected=true` through
  `qq:1049511700:qq`; all six report `status=warn` with missing
  text/image/file blockers because the runtime was just restarted and no fresh
  observe-only media samples had arrived yet.
- Verify runtime overview and dashboard summary include observe capture fields.
- Current local state is expected to warn until each observe group has at least
  one text sample, image asset, and file asset.

## Follow-up

- Ask the operator to produce one text message, one image, and one file in an
  observe-only QQ group, then use this endpoint to verify the full chain.
- Once image/file observe capture is verified, consider adding QQ receiver lease
  gating if duplicate local QQ receiver processes become a real risk.
