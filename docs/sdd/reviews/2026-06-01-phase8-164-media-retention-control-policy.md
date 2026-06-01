# Review: Media Retention Control Policy

Spec: `docs/sdd/specs/agent-gateway/107-media-retention-control-policy.md`

Implementation summary:
- Added `media_asset_retention / cleanup_expired` to the Go domain control mutation policy.
- Added policy API tests for listing/filtering the new target.
- Added preflight tests proving active approval can pass for `cleanup_expired` and unsupported media retention actions remain blocked.

Tests run:
- `go test ./domain/service ./app/service`

Findings:
- The change only extends deterministic policy/preflight recognition.
- No cleanup executor, approval creation, mutation creation, file deletion, metadata deletion, OCR/VLM, RAG, or AI call was added.

Decision:
- Accepted. The retention cleanup plan now has a matching allowlisted control mutation intent.

Follow-ups:
- A future cleanup executor must remain a separate SDD slice with approval binding, mutation audit recording, backup evidence, rate limits, and rollback checks.
