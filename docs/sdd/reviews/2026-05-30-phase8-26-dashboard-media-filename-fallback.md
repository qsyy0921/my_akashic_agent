# Review: dashboard media filename fallback

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:
- Kept dashboard media content access Go-first: `/api/dashboard/media-assets/content` still proxies to agent-runtime `/v1/media-assets/{asset_id}/content`.
- Added a compatibility fallback for stale dashboard state or old links where `asset_id` is actually a workspace upload filename.
- The fallback is limited to basename-only values and resolves the file under the current workspace `uploads` directory before serving it.
- Path traversal and Windows path forms continue to be rejected.

Tests run:
- `uv run pytest tests/test_dashboard_api.py::test_dashboard_media_asset_content_falls_back_to_workspace_upload_name tests/test_dashboard_api.py::test_dashboard_messages_are_enriched_with_runtime_media_assets -q --basetemp .tmp/pytest-dashboard-media-fallback`

Findings:
- Existing QQ media assets are correctly registered in Go, but cached or stale frontend rows can still request the dashboard media proxy with a filename as `asset_id`.
- Without the fallback, those requests show `media asset not found` or a broken image even though the file exists in `.akashic-workspace/uploads`.

Decision:
- Accept this as a narrow compatibility fix. Go remains the authoritative media registry for current rows; dashboard fallback only covers legacy filename-shaped references inside workspace uploads.

Follow-ups:
- Add Go/Python media content contract fixtures so stale filename links, valid Go asset ids, and forbidden paths stay covered together.
