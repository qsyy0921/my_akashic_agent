# Review: Phase 8.3 Go Media Registry Persistence

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:
- Added `infrastructure/mediaassetstore.Store`, a file-backed implementation of
  the Go `MediaAssetRepository` port.
- Added startup selection through `AKASHIC_MEDIA_ASSETS_DSN` and
  `AKASHIC_MEDIA_ASSETS_PATH`; the special value `memory` preserves development
  in-memory behavior.
- Wired the selected repository into both message ingest attachment registration
  and `/v1/media-assets` query/register/content APIs, keeping one authoritative
  registry.

Tests run:
- `go test ./...` under `services/agent-runtime`
- `uv run pytest tests/test_shadow_gateway.py tests/test_bootstrap_wiring_p2.py::test_config_load_reads_shadow_runtime_integration_block_with_compatibility -q --basetemp .tmp/pytest-media-persist`
- `git diff --check -- services\agent-runtime docs\sdd`

Findings:
- JSON persistence matches the existing `agentjobstore` pattern and keeps the
  app/domain layers independent from file IO.
- Store reload validates records and skips malformed assets rather than failing
  the runtime because of one corrupt record.

Decision:
- Approved as a Go-owned infrastructure slice. Media registry state now survives
  runtime restarts when the file-backed store is configured.

Follow-ups:
- Point dashboard attachment links at the Go content route.
- Consider SQLite once message/media/event volumes outgrow JSON file updates.
