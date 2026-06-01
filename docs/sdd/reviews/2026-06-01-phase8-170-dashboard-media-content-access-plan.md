# Phase 8 Review 170: Dashboard Media Content Access Plan

## Scope

- Connect Python dashboard media content proxy to Go-owned `content-access-plan` as an error fallback.
- Preserve existing successful Go content proxying and workspace-upload fallback behavior.
- Return structured access diagnostics when content cannot be displayed.

## Design Check

- Go remains source of truth for deterministic media access policy and blockers.
- Python dashboard only proxies bytes, serves legacy local upload fallback, or presents Go's access-plan detail.
- The change does not add OCR, VLM, file parsing, RAG, AI, remote media download, policy mutation, metadata mutation, or file deletion.

## Verification

- `uv run pytest tests\test_dashboard_api.py -q`
- `go test ./...` from `services/agent-runtime`
- `git diff --check`

## Risk

- If Go runtime is unreachable, the dashboard keeps the previous 502 behavior.
- If Go content route fails and the access-plan call also fails, the dashboard still returns the original upstream error.

## Result

Accepted. The frontend-facing content path now exposes Go-owned diagnostics without moving content policy into Python.
