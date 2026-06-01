# SPEC-109: Media Retention Metadata Cleanup

## Status

Accepted for the current iteration.

## Context

The Go runtime already exposes:

- media asset retention diagnostics;
- media asset retention cleanup plan;
- media-specific cleanup preflight that validates active operator approval and
  returns suggested control mutation audit metadata.

The next safe step is not physical file deletion. Akashic still needs stable
visibility for QQ/Telegram attachments, and file paths may point to external
NapCat or browser caches. The first executor should only remove expired media
asset registry metadata from the Go runtime after approval, while preserving
local files for rollback/manual inspection.

## Boundary Analysis

Go owns:

- deterministic retention candidate selection;
- approval/preflight gate;
- Go media registry metadata deletion;
- control mutation audit records for applied/failed cleanup;
- HTTP API contract, dry-run behavior, and side-effect declaration.

Python owns:

- OCR/VLM/image/file parsing;
- semantic memory/RAG extraction from attachments;
- AI decisions about whether content is useful.

Out of scope:

- deleting local files or remote provider files;
- changing NapCat/Telegram caches;
- calling Python, MQ, OCR/VLM, RAG, or AI;
- automatic scheduled cleanup.

## Decision

Add a metadata-only cleanup executor:

```text
POST /v1/media-assets/retention-cleanup
```

Request fields mirror retention filters and require:

- `target_id`
- `operator_id`
- `approval_id`
- optional `mutation_id`
- optional `dry_run`

Behavior:

1. Build the same cleanup candidate plan used by retention diagnostics.
2. Run media cleanup preflight with
   `target_kind=media_asset_retention` and `action=cleanup_expired`.
3. If `dry_run=true`, return candidates and no mutation audit is recorded.
4. If preflight is not ready, return blockers and do not delete metadata.
5. If ready and not dry-run, delete only the candidate records from the Go
   media asset repository.
6. Record an `applied` control mutation audit when all candidate metadata
   records are removed; record `failed` if any deletion fails.

## Safety Invariants

- No physical file deletion.
- No approval or mutation audit is created by GET/preflight endpoints.
- POST apply requires active approval through the existing preflight path.
- Permanent/non-expired assets are never selected by the executor.
- Failed preflight and dry-run produce no metadata deletion.

## Acceptance

- Repository delete is implemented for memory and JSON stores.
- Service tests cover dry-run, missing approval, and successful metadata-only
  cleanup with applied audit.
- HTTP tests cover POST apply and method guard.
- Full Go regression passes.
- TODO is cleared at the end of the iteration.
