# SPEC-115: Dashboard Message Media Access Plan Link

## Status

Accepted for the current iteration.

## Context

Dashboard messages are enriched with Go media asset metadata and a
`content_url`. The dashboard now also exposes
`/api/dashboard/media-assets/content-access-plan`, but message list/detail
responses do not include a direct link to that plan. The frontend has to
reconstruct the URL or open the content route first.

## Boundary Analysis

Go owns:

- media asset metadata and content access policy;
- content access plan readiness, blockers, and endpoint.

Python dashboard owns:

- message presentation enrichment;
- stable browser-facing links to Go-backed dashboard proxy routes.

Out of scope:

- Python fetching access-plan detail for every message row;
- Python duplicating content policy or blockers;
- content streaming, remote download, OCR/VLM/RAG/AI, or file parsing.

## Decision

When dashboard message enrichment attaches `media_assets`, add:

```json
"content_access_plan_url": "/api/dashboard/media-assets/content-access-plan?asset_id=..."
```

for each asset that has an `asset_id`. Preserve existing `content_url` behavior.
Do not fetch the access plan during list/detail enrichment; this remains a link
to a read-only Go-owned diagnostic endpoint.

## Acceptance

- Message list and detail media assets include both `content_url` and
  `content_access_plan_url`.
- Existing message media asset enrichment behavior remains unchanged.
- Target dashboard API tests pass, Go regression still passes, and TODO is
  cleared.
