# Phase 8.138 Review: Runtime Overview Media Asset Content

## Scope

Aggregated Go-owned media asset content diagnostics into
`/v1/runtime-overview` so dashboard/operator entrypoints can see attachment
display health without discovering `/v1/media-assets/content-diagnostics`
manually.

## Changes

- Added optional `RuntimeOverviewDeps.MediaAssetContentDiagnostics`.
- Added `RuntimeOverviewView.media_asset_content_diagnostics`.
- Added summary fields:
  - `media_asset_content_assets`
  - `media_asset_content_ready`
  - `media_asset_content_forbidden`
  - `media_asset_content_unavailable`
  - `media_asset_content_disabled`
  - `media_asset_content_error`
- Added `Media Asset Content` runtime overview card with `ready/assets` value.
- Wired `cmd/agent-runtime` to reuse the existing media asset service.
- Updated README, SDD index, DONE, LIVE_CHECKS, and BACKLOG.

## Boundary Check

Go remains responsible for deterministic attachment infrastructure:

- safe content access readiness;
- card/summary/detail aggregation;
- runtime API stability.

Python remains responsible for AI media processing:

- OCR, VLM, file parsing;
- image generation;
- media-to-memory/RAG semantic extraction;
- provider-specific fallback and prompt strategy.

The change does not mutate media asset metadata, content roots, `/content`
authorization, queues, AgentJob state, or Python worker state.

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Risks

- Runtime overview now performs bounded content diagnostics through the existing
  media asset service. The limit is capped, but very slow storage roots can
  still affect dashboard latency.
- A `ready` attachment only proves local content access, not successful OCR/VLM
  or semantic extraction.
