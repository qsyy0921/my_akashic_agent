# Phase 8 Review 169: Media Asset Content Access Plan

## Scope

- Add Go-owned `GET /v1/media-assets/content-access-plan?asset_id=...`.
- Return deterministic access readiness, reason, blockers, content endpoint, content mime/size when safely probed, and operator-facing steps.
- Keep Python responsible for OCR, VLM, file parsing, semantic extraction, RAG, and AI summaries.

## Design Check

- DDD boundary remains intact: query view in `app/query`, use case in `app/service`, inbound contract in `app/port/in`, HTTP adapter in `trigger/http`.
- The endpoint is read-only and uses the same deterministic local content reader policy as content diagnostics.
- The endpoint never streams bytes to the caller, downloads remote media, changes media roots, deletes metadata/files, or triggers Python AI work.

## Verification

- `go test ./app/service ./trigger/http`
- `go test ./...`
- `git diff --check`

## Risk

- The plan probes local content by opening and immediately closing it, matching existing diagnostics behavior. Very large files are not read into memory.
- Remote URL media remains unavailable until a downloader/cache worker stores a local path; that remains a separate infrastructure concern.

## Result

Accepted. The change improves frontend/debug visibility for QQ/Telegram attachments while preserving Go/Python responsibility boundaries.
