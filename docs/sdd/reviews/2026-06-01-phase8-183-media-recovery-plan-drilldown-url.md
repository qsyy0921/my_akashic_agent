# Review: Media Recovery Plan Drilldown URL

## Scope

- Add deterministic recovery-plan drilldown links to Go media content
  diagnostics and Python dashboard message media assets.
- Keep list/detail enrichment read-only and avoid fetching the recovery plan
  during rendering.

## Result

- `MediaAssetContentDiagnosticItemView` now includes
  `content_recovery_plan_endpoint`.
- Dashboard media asset enrichment now adds `content_recovery_plan_url` beside
  `content_url` and `content_access_plan_url`.
- Dashboard TypeScript media asset type declares both plan URL fields.

## Boundary

Go continues to own deterministic media asset runtime URLs and diagnostics.
Python only projects browser-friendly links and does not copy media access
policy. OCR, VLM, file parsing, RAG and AI execution remain Python AI worker
responsibilities and are not triggered by these fields.

## Tests

- `go test ./app/service ./trigger/http`
- `uv run pytest tests/test_dashboard_api.py -q` with `TMP/TEMP` redirected to
  `.tmp/pytest`.
