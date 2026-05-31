# Runtime Overview Media Asset Content

## Context

`080-media-asset-content-diagnostics.md` added a read-only media asset content
diagnostics endpoint. It is useful for a single asset or filtered asset list, but
the main dashboard entrypoint is `/v1/runtime-overview`; without aggregation,
operators still need to know the separate endpoint before they can understand why
QQ group images or files fail to render.

## Boundary

Go owns deterministic media asset infrastructure:

- media asset metadata and safe local content access policy;
- content readiness classification: `ready`, `forbidden`, `unavailable`,
  `disabled`, and `error`;
- runtime overview summary/card/detail fields;
- read-only dashboard diagnostics.

Python keeps AI and experimental media semantics:

- OCR, VLM and file content parsing;
- image generation and multimodal tool execution;
- semantic memory/RAG extraction from media content;
- provider-specific fallback and prompt strategy.

This slice must not change `/v1/media-assets/{asset_id}/content` authorization
or storage roots. It only aggregates the existing diagnostics result.

## Design

Add an optional `RuntimeOverviewDeps.MediaAssetContentDiagnostics` dependency
with the same application-level capability as the media asset manager:

```text
RuntimeOverviewService
  -> MediaAssetContentDiagnostics.ContentDiagnostics(limit)
  -> RuntimeOverviewView.MediaAssetContentDiagnostics
  -> Summary:
       media_asset_content_assets
       media_asset_content_ready
       media_asset_content_forbidden
       media_asset_content_unavailable
       media_asset_content_disabled
       media_asset_content_error
  -> Card:
       id: media_asset_content
       value: ready/assets
       status:
         danger if forbidden/unavailable/error > 0
         warn if disabled > 0
         ok if ready > 0
         muted if no assets
```

`ContentDiagnostics` remains read-only. It may open content only to verify
availability and must close it immediately, matching the existing endpoint.

## HTTP Shape

`GET /v1/runtime-overview` includes:

```json
{
  "summary": {
    "media_asset_content_assets": 4,
    "media_asset_content_ready": 1,
    "media_asset_content_forbidden": 1,
    "media_asset_content_unavailable": 1,
    "media_asset_content_disabled": 1,
    "media_asset_content_error": 0
  },
  "cards": [
    {
      "id": "media_asset_content",
      "label": "Media Asset Content",
      "value": "1/4",
      "status": "danger"
    }
  ],
  "media_asset_content_diagnostics": {
    "side_effect": "none"
  }
}
```

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Risks

- The aggregate can be expensive if a very large media repository is scanned.
  The service uses runtime overview's bounded limit and the existing diagnostics
  limit cap.
- The status only proves deterministic content access. A `ready` asset may still
  fail OCR/VLM or semantic parsing in Python.
