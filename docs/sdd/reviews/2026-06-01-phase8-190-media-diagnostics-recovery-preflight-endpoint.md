# Review: Media Diagnostics Recovery Preflight Endpoint

## Scope

- Add `content_recovery_preflight_endpoint` to Go media content diagnostics.
- Render a preflight link in the runtime overview `Media Asset Content` table.
- Update the media content contract fixture.

## Result

- `MediaAssetContentDiagnosticItemView` now includes the Go-owned recovery
  preflight endpoint.
- `ContentDiagnostics` populates the endpoint deterministically from `asset_id`.
- Runtime overview dashboard displays `content`, `access`, `recovery`, and
  `preflight` links from already loaded card detail.

## Boundary

Go remains the source of truth for media asset endpoint hints and recovery
preflight semantics. Dashboard rendering is read-only and performs no preflight
requests. This slice does not create approvals or mutation audits, download,
restore, cache, stream, parse, or invoke OCR/VLM/RAG/AI.

## Tests

- `npm run build:plugins`
- `npm run typecheck`
- `go test ./...`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_sdd_contract_fixtures.py -q`
- `git diff --check`
