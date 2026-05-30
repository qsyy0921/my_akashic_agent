# Review: Phase 8.21 Media Safe-Root Discovery

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Problem:
- Dashboard attachment links proxied to Go returned `403 media asset content
  forbidden` for QQ images that were already mirrored into
  `.akashic-workspace/uploads`.
- The file existed, but `agent-runtime` had been launched from the repository
  root. The old default used `cwd/../..` as the repo root, which incorrectly
  resolved the safe root outside `E:\agent\akashic`.

Implementation summary:
- Replaced the fixed `cwd/../..` default with upward Akashic root discovery from
  both the current working directory and the executable directory.
- Default local media roots now remain correct when the runtime starts from the
  repository root, `services/agent-runtime`, or `.tmp/bin`.
- Kept `AKASHIC_MEDIA_ASSET_ROOTS` as the explicit production override.
- Added focused Go tests for repository-root and binary/service-directory
  startup layouts.

Tests run:
- `go test ./...` under `services/agent-runtime`
- `uv run pytest tests/test_shadow_audit_plugin.py::test_dashboard_media_asset_content_proxy_uses_agent_runtime tests/test_dashboard_api.py::test_dashboard_messages_are_enriched_with_runtime_media_assets -q --basetemp .tmp\pytest-media-content`
- Live smoke: `GET /v1/media-assets/{asset_id}/content` returned `200
  image/gif` for a QQ mirrored attachment.
- Live smoke: `GET /api/dashboard/media-assets/content?asset_id=...` returned
  `200 image/gif` and `200 image/jpeg` for two QQ mirrored attachments.

Decision:
- Approved. This is a runtime startup inference bug, not a relaxation of the
  content boundary. Go still serves only registered local media that resolves
  inside the discovered or explicitly configured safe roots.

Follow-ups:
- Add Go/Python contract fixtures that include `/v1/media-assets/{id}/content`
  happy-path and forbidden-path cases.
