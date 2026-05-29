# Review: Phase 5.1 Go Media Registry Metadata

Spec:

- `docs/sdd/specs/message-gateway/007-media-file-registry.md`
- `docs/sdd/adr/0003-go-service-ddd-granularity.md`

Implementation summary:

- Added Go `MediaAsset` domain model for account-aware image/file metadata.
- Added app-layer command/query/ports/service for registering and querying
  media assets.
- Added in-memory media asset repository to the existing gateway store.
- Added `/v1/media-assets` metadata API.
- Kept `/v1/media-assets/{asset_id}/content` disabled with `501 Not
  Implemented` until path validation and access policy are reviewed.
- Kept all Go code under `services/message-gateway`; no new service split.

Tests run:

- `gofmt -w api app cmd domain infrastructure trigger types`
- `go test ./...`
- `go build ./cmd/message-gateway`
- `uv run pytest tests\test_sdd_contract_fixtures.py tests\test_shadow_gateway.py -q`
- `git diff --check`

Findings:

- The registry is metadata-only and in-memory. It is suitable as a control-plane
  slice, not yet durable production asset storage.
- Python QQ attachment mirroring is not wired to the new API yet.
- Dashboard links still need to switch from Python compatibility metadata to
  Go-owned asset ids in a later slice.

Decision:

- Accept as a Phase 5.1 metadata-only media registry slice.

Follow-ups:

- Mirror Python-captured QQ attachments into `/v1/media-assets`.
- Add persistence for media metadata.
- Add controlled content route after workspace path and signed URL policy tests.
- Add dashboard media panel/query against Go registry.
