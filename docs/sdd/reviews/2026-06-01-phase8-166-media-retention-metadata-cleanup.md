# Review: Media Retention Metadata Cleanup

Spec: `docs/sdd/specs/agent-gateway/109-media-retention-metadata-cleanup.md`

Implementation summary:
- Added `DeleteMediaAsset` to the Go media asset repository port and implemented it for memory and JSON stores.
- Added a metadata-only media retention cleanup service behind the existing preflight path.
- Added `POST /v1/media-assets/retention-cleanup` with dry-run and apply behavior.
- Added service, HTTP, and persistent store tests.

Tests run:
- `go test ./app/service ./trigger/http ./infrastructure/mediaassetstore`
- `go test ./...`
- `git diff --check`

Findings:
- Apply requires the media-specific cleanup preflight to be ready.
- Dry-run and failed preflight do not delete metadata and do not record mutation audit.
- Apply deletes only Go media asset metadata and records an applied control mutation audit.
- No local file deletion, provider cache mutation, Python call, OCR/VLM, RAG, MQ, or AI execution was added.

Decision:
- Accepted. This gives operators a controlled metadata cleanup path while keeping physical attachment deletion out of scope.

Follow-ups:
- Physical file cleanup, backup bundles, rate-limited scheduled cleanup, and rollback restore remain separate SDD slices.
