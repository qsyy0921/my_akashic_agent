# Review: Phase 8.1 Go Media Content Access

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:
- Added a Go `MediaAssetContentReader` outbound port and `OpenContent` app
  use case.
- Added `infrastructure/localmedia.Reader` to serve only registered local files
  under configured safe roots.
- Enabled `GET /v1/media-assets/{asset_id}/content` for safe local files with
  MIME type, content length, and inline content disposition headers.
- Wired local runtime defaults to Akashic workspace upload directories, with
  `AKASHIC_MEDIA_ASSET_ROOTS` as the explicit override.

Tests run:
- `go test ./...` under `services/agent-runtime`
- `uv run pytest tests/test_shadow_gateway.py tests/test_bootstrap_wiring_p2.py::test_config_load_reads_shadow_runtime_integration_block_with_compatibility -q --basetemp .tmp/pytest-media-runtime`
- `git diff --check -- services\agent-runtime docs\sdd`

Findings:
- Content access is intentionally local-only. Remote platform URLs must first be
  mirrored into a safe media root before Go will serve them.
- The route rejects files outside safe roots with `403` and missing or
  unsupported local content with `404`.

Decision:
- Approved as a Go-owned infrastructure slice. Python remains responsible for
  platform download/OCR/VLM workers, while Go now owns the stable content access
  boundary for registered local media.

Follow-ups:
- Mirror QQ observed attachments into `/v1/media-assets` automatically.
- Persist media registry metadata beyond the in-memory development store.
- Point the dashboard attachment links at the Go content route.
