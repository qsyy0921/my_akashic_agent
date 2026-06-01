# Phase 8 Review 171: Dashboard Media Content Access Plan Proxy

## Scope

- Add `/api/dashboard/media-assets/content-access-plan?asset_id=...`.
- Proxy Go `/v1/media-assets/content-access-plan` for direct frontend diagnostics.
- Preserve existing media content proxy and fallback behavior.

## Design Check

- Go remains the source of truth for media content access policy, blockers, endpoint, and side effect.
- Python dashboard only exposes a read-only browser-facing proxy and normalizes runtime transport errors.
- No content streaming, remote media download, OCR, VLM, file parsing, RAG, AI, metadata mutation, or file deletion is added.

## Verification

- `uv run pytest tests\test_dashboard_api.py -q`
- `go test ./...` from `services/agent-runtime`
- `git diff --check`

## Risk

- If Go runtime is not configured, the endpoint returns 503 like other runtime-backed dashboard APIs.
- If Go returns an invalid plan payload, the dashboard returns 502 instead of inventing a fallback policy.

## Result

Accepted. The frontend can now query Go-owned attachment access diagnostics before opening the content route.
