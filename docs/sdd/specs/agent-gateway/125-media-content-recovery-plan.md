# SPEC-125: Media Content Recovery Plan

## Status

Accepted for the current iteration.

## Context

Media assets can now be diagnosed and proxied, but when content is disabled,
forbidden, unavailable, or errors during probing, operators only get the raw
content access reason. `OPEN_ISSUES.md` tracks this as OI-005: remote
redownload, file restore, and content cache strategy are not unified yet.

Before implementing a downloader/cache executor, Go should expose a deterministic
read-only recovery plan that explains what to inspect, what can be retried, and
which future executor would own the side effect.

## Boundary Analysis

Go owns:

- media asset metadata and deterministic content access status;
- read-only recovery plan shape, URL hints, and operator steps;
- future control-plane boundary for cache/download/restore actions.

Python owns:

- OCR, VLM, file parsing, semantic extraction, RAG, and AI interpretation of the
  media content.

Out of scope:

- downloading or redownloading remote media;
- restoring files;
- mutating media metadata;
- opening/streaming file content beyond the existing probe;
- OCR/VLM/RAG/AI execution.

## Decision

Add `GET /v1/media-assets/content-recovery-plan?asset_id=...`.

The endpoint delegates to the existing access-plan probe and returns:

- `ready`: true only when no recovery is needed;
- `reason`: `media_asset_content_recovery_not_required` or a stable
  `media_asset_content_recovery_*` reason;
- runtime/dashboard URL hints;
- access plan snapshot;
- required/verify/fallback steps;
- `side_effect=none`.

Python dashboard adds a matching read-only proxy at
`GET /api/dashboard/media-assets/content-recovery-plan?asset_id=...` so browser
links can inspect the Go plan without copying media access policy.

## Acceptance

- Go service and HTTP tests cover ready, unavailable, forbidden, disabled, and
  method-not-allowed behavior.
- Python dashboard tests cover the read-only recovery-plan proxy and invalid
  upstream response handling.
- Response explicitly does not download, restore, parse, OCR/VLM, RAG, or invoke
  AI.
- OI-005 is updated to mention the new read-only recovery plan while keeping the
  real downloader/cache executor open.
- TODO is cleared at the end of the iteration.
