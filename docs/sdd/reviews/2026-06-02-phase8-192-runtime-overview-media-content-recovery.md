# Phase 8 Review 192: Runtime Overview Media Content Recovery

## Scope

- Added `media_asset_content_recovery` to Go runtime overview.
- Aggregates `media_asset_content/recover_content` control mutation audits into
  summary fields, card status/value, and a focused detail object.
- Updated SDD project status, DONE, LIVE_CHECKS, spec index, and review index.

## Boundary Review

- Go only reads existing control mutation audit state.
- The overview does not call `POST /v1/media-assets/content-recovery`, create
  approvals, create mutation audits, download files, update media registry, or
  enqueue Python jobs.
- Python remains responsible for OCR/VLM/file parsing, semantic memory, RAG, and
  provider-specific private source recovery.

## Verification

- `go test ./app/service`

## Residual Risk

- Python dashboard may still render this new detail as raw JSON until a separate
  dashboard drilldown table is added.
- OI-005 remains open for platform-private media recovery and post-recovery AI
  enrichment admission.
