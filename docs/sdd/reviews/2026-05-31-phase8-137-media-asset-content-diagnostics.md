# Phase 8.137 Media Asset Content Diagnostics

Spec: `docs/sdd/specs/agent-gateway/080-media-asset-content-diagnostics.md`

## Changes

- Added `GET /v1/media-assets/content-diagnostics`.
- Added `MediaAssetContentDiagnosticsFilter`,
  `MediaAssetContentDiagnosticItemView` and
  `MediaAssetContentDiagnosticsView`.
- Extended `MediaAssetManager` and `MediaAssetService` with a read-only
  `ContentDiagnostics` use case.
- The endpoint classifies content access as `ready`, `forbidden`,
  `unavailable`, `disabled`, or `error`, and returns the stable content route
  for dashboard use.

## Boundary Review

- Go owns deterministic media asset metadata, safe local content access policy,
  content readiness checks and stable runtime API shape.
- Python still owns OCR, VLM, file parsing, semantic extraction, multimodal
  prompt construction and downstream Memory/RAG/image pipelines.
- This slice does not change `/v1/media-assets/{asset_id}/content` access
  policy and does not download remote platform URLs.

## Verification

- `go test ./app/service -run TestMediaAssetServiceContentDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestMediaAssetEndpointRegistersListsAndServesContentRoute -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Residual Risk

- Diagnostics can only classify registered assets. If a platform adapter fails
  before registration, observe-capture diagnostics and receiver logs are still
  needed.
- Remote platform URLs must still be mirrored into a configured safe root before
  Go can serve content.
