# Review: primary dashboard message media assets

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:
- Extended Go `agent-runtime` media asset listing with DDD-style
  `MediaAssetFilter`.
- Added route/source filters to `/v1/media-assets`, including
  `channel_kind`, `conversation_id`, `conversation_type`,
  `source_message_id`, `source_message_id_suffix`, and `asset_kind`.
- Preserved `platform_message_id` and `session_message_id` in media asset
  metadata during message ingest for newer observed messages.
- Enriched primary dashboard message rows with `media_assets` by querying Go
  using message platform metadata.
- Updated React attachment rendering to prefer `media_assets[].content_url` and
  use legacy workspace upload paths only as fallback.

Tests run:
- `npm run typecheck`
- `npm run build`
- `uv run pytest tests\test_dashboard_api.py::test_dashboard_messages_are_enriched_with_runtime_media_assets tests\test_dashboard_api.py::test_dashboard_attachment_endpoint_serves_workspace_uploads tests\test_shadow_audit_plugin.py tests\test_agent_jobs_dashboard_plugin.py tests\test_bootstrap_wiring_p2.py::test_config_load_reads_shadow_runtime_integration_block_with_compatibility -q --basetemp .tmp\pytest-dashboard-message-media-final`
- `C:\Users\qsyy0921\.codex\tools\go1.26.3\bin\go.exe test ./...`
- `git diff --check -- bootstrap frontend plugins services docs\sdd tests static\dashboard`
- Live smoke: rebuilt and restarted `agent-runtime`, restarted dashboard,
  verified `/v1/media-assets?...source_message_id_suffix=...` returns the QQ
  image asset, verified `/api/dashboard/messages` includes `media_assets`, and
  verified the dashboard content proxy returns `image/jpeg` bytes.

Findings:
- Existing stored messages often do not know the bot account id, so dashboard
  lookup uses conversation plus platform message id suffix instead of requiring
  exact source event id.
- The Go registry remains the authority for serving bytes; Python only performs
  read-side enrichment and same-origin URL assembly.

Decision:
- Accepted as a migration step toward Go-owned media infrastructure. It moves
  the primary dashboard away from filesystem path contracts without breaking old
  rows that do not yet have registered assets.

Follow-ups:
- Add a true batch resolve endpoint if dashboard pages start making too many
  per-message media asset queries.
- Backfill older message rows with asset ids once the group message archive is
  migrated into Go-owned storage.
