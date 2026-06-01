# Phase 8 Review 174: Media Content Diagnostics Access Plan Endpoint

## Scope

- Add `content_access_plan_endpoint` to Go media content diagnostics items.
- Normalize the field in Python runtime overview dashboard, with fallback for older Go runtime payloads.
- Keep access-plan detail lookup lazy; diagnostics do not fetch plans for every item.

## Design Check

- Go remains owner of runtime API paths and deterministic media content status.
- Python only preserves or deterministically derives the Go endpoint for presentation.
- No content streaming, remote media download, OCR, VLM, RAG, AI, metadata mutation, or file deletion is added.

## Verification

- `go test ./app/service ./trigger/http`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q`
- `go test ./...`
- `git diff --check`

## Risk

- Older Go runtimes will not return the field; dashboard fallback derives it from `asset_id`.
- Consumers that ignore the new field continue using `content_endpoint` unchanged.

## Result

Accepted. Batch media content diagnostics now link to single-asset access plans without moving policy into Python.
