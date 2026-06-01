# Review: Media Content Recovery Plan

## Scope

- Add Go-owned `GET /v1/media-assets/content-recovery-plan?asset_id=...`.
- Add Python dashboard proxy
  `/api/dashboard/media-assets/content-recovery-plan?asset_id=...`.
- Keep recovery planning read-only: no download, restore, stream, OCR/VLM, RAG,
  file parsing, metadata mutation, or AI execution.

## Result

- Go service builds recovery plans from the existing content access plan, mapping
  ready/disabled/forbidden/unavailable/error states to stable
  `media_asset_content_recovery_*` reasons.
- Response includes runtime/dashboard URL hints, access plan snapshot,
  required/verify/fallback steps, optional future executor scope, and
  `side_effect=none`.
- OI-005 remains open for a future operator-approved downloader/cache executor.

## Tests

- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests/test_dashboard_api.py -q` with `TMP/TEMP` redirected to
  `.tmp/pytest` because the default Windows pytest temp root was permission
  denied.
