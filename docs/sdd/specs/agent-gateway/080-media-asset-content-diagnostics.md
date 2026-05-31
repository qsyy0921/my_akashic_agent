# 080 Media Asset Content Diagnostics

Date: 2026-05-31

## Problem

QQ group images and files are registered as Go `MediaAsset` records and served
through `/v1/media-assets/{asset_id}/content`. When the browser cannot display a
file, the current failure path only exposes an HTTP status such as 403 or 404.
Operators and the dashboard need a stable read-only diagnostic endpoint that
can explain whether content is ready, forbidden by the configured safe roots,
unavailable on disk, or disabled because no content reader is configured.

This is deterministic runtime infrastructure and belongs in Go. It should not
execute OCR, VLM, file understanding, image generation, or RAG ingestion.

## Scope

- Add `GET /v1/media-assets/content-diagnostics`.
- Reuse the existing media asset repository and content reader.
- Support the existing media asset filters plus optional `asset_id`.
- Return per-asset status and summary totals:
  - `ready`
  - `forbidden`
  - `unavailable`
  - `disabled`
  - `error`
- Include the stable content route for frontends:
  `/v1/media-assets/{asset_id}/content`.
- Keep the endpoint read-only. Opening content is only used for availability
  checking and any opened body is closed immediately.

## Go / Python Boundary

Go owns:

- media asset registration and deterministic metadata;
- safe local content access policy;
- content readiness diagnostics;
- dashboard/runtime API shape and tests.

Python owns:

- image OCR/VLM interpretation;
- file parsing and semantic extraction;
- multimodal prompt construction;
- memory/RAG/image-generation AI pipelines.

## Non-goals

- No OCR/VLM/file parsing.
- No media download retry.
- No upload to RAGFlow or vector store.
- No change to existing `/content` access policy.
- No broad dashboard UI redesign in this slice.

## API

```text
GET /v1/media-assets/content-diagnostics?limit=50
GET /v1/media-assets/content-diagnostics?asset_id=asset%3Aqq%3A...
GET /v1/media-assets/content-diagnostics?channel_kind=qq&conversation_id=27234224&asset_kind=image
```

Example response data:

```text
{
  "items": [
    {
      "asset_id": "asset:qq:...",
      "kind": "image",
      "content_status": "forbidden",
      "content_reason": "media_asset_content_forbidden",
      "content_endpoint": "/v1/media-assets/asset%3Aqq%3A.../content"
    }
  ],
  "totals": {
    "assets": 1,
    "forbidden": 1
  },
  "side_effect": "none"
}
```

## Acceptance

- Service tests cover ready, disabled, forbidden and unavailable diagnostics.
- HTTP route test verifies JSON shape and URL-escaped content endpoint.
- README documents the endpoint and non-goals.
- SDD review records tests and residual risks.
