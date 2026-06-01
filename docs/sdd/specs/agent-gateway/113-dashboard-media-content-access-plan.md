# SPEC-113: Dashboard Media Content Access Plan Fallback

## Status

Accepted for the current iteration.

## Context

The browser opens QQ/Telegram media through
`/api/dashboard/media-assets/content?asset_id=...`, which proxies Go
`/v1/media-assets/{asset_id}/content`. When the Go content endpoint returns
403/404/501 and the old workspace-upload fallback cannot find a local file, the
browser currently sees only a raw error. Go now exposes a deterministic
`/v1/media-assets/content-access-plan` endpoint that explains why content is or
is not accessible.

## Boundary Analysis

Go owns:

- media asset metadata;
- content access policy and local-root checks;
- ready/reason/blockers/content endpoint plan.

Python dashboard owns:

- proxying content bytes when Go content is ready;
- legacy workspace upload fallback for existing local files;
- read-only presentation of Go access-plan diagnostics.

Out of scope:

- Python reimplementing content root policy;
- Python downloading remote media;
- OCR/VLM/file parsing/RAG/AI;
- modifying access policy, metadata, or files.

## Decision

When `/api/dashboard/media-assets/content` receives an HTTP error from Go
content route:

1. Try the existing workspace-upload fallback for 403/404 compatibility.
2. If fallback is unavailable, query Go
   `/v1/media-assets/content-access-plan?asset_id=...`.
3. Return JSON error detail containing the upstream message and normalized
   `content_access_plan`.

The dashboard remains a read-only adapter; Go remains the source of truth.

## Acceptance

- Existing successful proxy and workspace fallback tests continue to pass.
- New test covers 403 with no local fallback and verifies plan detail is
  returned.
- Calls to the plan endpoint do not trigger AI or content parsing.
- Target Python test passes, Go regression still passes, and TODO is cleared.
