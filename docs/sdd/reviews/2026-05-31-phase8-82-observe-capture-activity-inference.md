# Review: observe capture activity inference

## Scope

- Enhanced Go observe capture diagnostics to infer effective receiver
  connectivity from recent observe-only inbox events.
- Added separate API fields for status heartbeat connectivity and recent
  activity connectivity.
- Added runtime overview summary counters for observe-capture receiver
  connectivity sources.
- Updated dashboard runtime overview normalization to preserve the new fields.
- Updated the SDD index and Chinese migration TODO.

## Boundaries

- No QQ/Telegram sends were executed.
- Receiver status and receiver lease state remain unchanged by this diagnostic
  inference.
- This does not mark file coverage as complete; groups with no file assets still
  warn on `file_not_seen`.

## Validation

- `go test ./...`
- `go vet ./...`
- `go build -o ..\..\.tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-observe-capture-activity`
- `uv run python -m py_compile plugins\runtime_overview\dashboard.py`

## Runtime Smoke

- Rebuilt and restarted local `agent-runtime` on `:8780`.
- `GET /v1/observe-capture-diagnostics` returned 6 observe-only QQ group
  targets, 310 inbox events, 54 image assets, and 54 content-ready assets.
- 4 targets with recent inbox activity reported `receiver_connected=true` with
  `receiver_connection_source=recent_inbox_activity`; their only remaining
  blocker was `file_not_seen`.
- 2 targets without recent inbox activity still reported
  `receiver_not_connected`, proving stale activity does not hide a disconnected
  receiver signal.
- `GET /v1/runtime-overview` summary exposed
  `observe_capture_receiver_connected=4`,
  `observe_capture_receiver_status_connected=0`, and
  `observe_capture_receiver_activity_recent=4`.
- No QQ/Telegram platform sends were executed.
