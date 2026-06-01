# Phase 8 Review 172: Dashboard Message Media Access Plan Link

## Scope

- Add `content_access_plan_url` to dashboard-enriched message `media_assets`.
- Preserve existing `content_url` behavior.
- Avoid fetching access-plan detail during message list/detail enrichment.

## Design Check

- Go remains the source of truth for media content access policy and blockers.
- Python dashboard only attaches stable browser-facing links to Go-backed dashboard routes.
- The change does not add content streaming, remote media download, OCR, VLM, file parsing, RAG, AI, metadata mutation, or file deletion.

## Verification

- `uv run pytest tests\test_dashboard_api.py -q`
- `go test ./...` from `services/agent-runtime`
- `git diff --check`

## Risk

- Frontend consumers that ignore the new field continue using `content_url` unchanged.
- The field is only present when Go media asset enrichment returns a stable `asset_id`.

## Result

Accepted. Message media assets now expose a direct read-only diagnostic link for attachment access state without moving access policy into Python.
