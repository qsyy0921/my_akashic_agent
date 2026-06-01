# SPEC-134: Media Content Recovery Executor

## Status

Accepted for the current iteration.

## Context

Go already owns media asset metadata, safe content access, content diagnostics,
read-only recovery plans, and approval-bound recovery preflight. OI-005 remains
open because missing QQ/Telegram images or files can be explained but cannot yet
be restored into the local content roots that the browser/dashboard can serve.

This slice adds a small Go-owned executor for deterministic media content
recovery. It is intentionally narrow: download a registered HTTP/HTTPS media
source into a configured local cache root, update the media asset registry, and
record a control mutation audit. AI interpretation of the restored content stays
in Python.

## Boundary Analysis

Go owns:

- media asset registry lookup and metadata update;
- approval-bound preflight for `media_asset_content/recover_content`;
- HTTP/HTTPS download into a configured cache root;
- local cache path/hash/size/mime metadata;
- control mutation audit records for applied/failed execution;
- HTTP API shape and deterministic error reasons.

Python owns:

- OCR, VLM, file parsing, image understanding, semantic memory, and RAG;
- provider-specific retry strategies that need cookies/session-specific browser
  state;
- deciding whether restored content should trigger AI enrichment jobs.

Out of scope:

- bypassing approval/preflight for actual writes;
- using QQ/Telegram private API credentials in Go;
- parsing recovered files or generating descriptions;
- deleting remote or local content;
- automatic background retries.

## Decision

Add:

```text
POST /v1/media-assets/content-recovery
```

Request body:

```json
{
  "asset_id": "asset:qq:...",
  "target_id": "optional, defaults to asset_id",
  "operator_id": "qsyy",
  "approval_id": "approval-...",
  "mutation_id": "optional audit id",
  "dry_run": false
}
```

Execution rules:

1. Always run the existing media content recovery preflight first.
2. `dry_run=true` returns preflight and planned execution metadata only; it does
   not download, write, update registry, or record mutation audit.
3. Actual execution requires preflight ready, a configured downloader/cache root,
   and a control mutation audit recorder.
4. The downloader accepts only `http` and `https` asset URLs.
5. Downloaded bytes are written under the configured recovery cache root using a
   deterministic safe filename derived from `asset_id`.
6. The media asset registry is updated with:
   - `metadata.local_path`;
   - `metadata.recovered_from_url`;
   - `metadata.recovered_at`;
   - `metadata.recovery_scope=download_to_local_cache`;
   - `metadata.recovery_mutation_id`;
   - `metadata.recovery_approval_id`.
7. MIME type, size, content hash, name, and `updated_at` are updated when known.
8. Applied or failed attempts record a control mutation audit.

## Failure Behavior

- Missing/blocked approval returns the preflight blockers and performs no IO.
- Unsupported source scheme returns `media_asset_content_recovery_source_unsupported`.
- Download/write failures return `media_asset_content_recovery_failed` and
  record a failed mutation audit when the audit recorder is available.
- Registry update failures also record a failed mutation audit.

## Observability

The response includes:

- `ready`, `applied`, `dry_run`, `reason`, `blockers`;
- target/action/operator/approval/mutation identifiers;
- recovered local path, mime type, size, content hash when applied;
- embedded preflight;
- applied/failed audit summaries.

## Acceptance

- Dry-run returns no side effects and does not require the downloader.
- Missing approval blocks actual execution before download/write.
- A successful HTTP download writes into the cache root, updates the registry,
  and records an applied control mutation audit.
- Unsupported source URL is blocked before writing content.
- HTTP handler accepts POST only and returns stable JSON.
- README, DONE, LIVE_CHECKS, OPEN_ISSUES, review, and spec index are updated.
- TODO is cleared after tests pass.
