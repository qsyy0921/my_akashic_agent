# Review: dashboard media links through agent-runtime

Spec:
- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`

Implementation summary:
- Added a same-origin dashboard proxy endpoint:
  `/api/dashboard/media-assets/content?asset_id=...`.
- The proxy forwards to Go `agent-runtime`:
  `/v1/media-assets/{asset_id}/content`.
- Shadow audit attachment normalization now adds `content_url` for attachments
  with asset ids.
- Shadow audit dashboard rendering prefers `content_url` over raw `url`, so QQ
  image/file links go through Go's safe media content policy.
- Dashboard plugin readers now prefer `AKASHIC_AGENT_RUNTIME_URL` /
  `AKASHIC_RUNTIME_BASE_URL` while retaining legacy gateway environment names.
- Fixed the Windows plugin build path to prefer the local esbuild Node entrypoint
  instead of directly spawning `.cmd` shims.

Tests run:
- `npm run build:plugins`
- `python -m compileall bootstrap\dashboard_api.py plugins\shadow_audit\dashboard.py plugins\agent_jobs\dashboard.py`
- `node --check scripts\build-plugins.mjs`
- `uv run pytest tests\test_shadow_audit_plugin.py tests\test_agent_jobs_dashboard_plugin.py tests\test_dashboard_api.py::test_dashboard_attachment_endpoint_serves_workspace_uploads -q --basetemp .tmp\pytest-dashboard-media-links-2`
- `C:\Users\qsyy0921\.codex\tools\go1.26.3\bin\go.exe test ./...`
- `git diff --check -- bootstrap plugins scripts tests docs\sdd`
- Live smoke: restarted `uv run python main.py --workspace .akashic-workspace`,
  verified `/api/dashboard/shadow-audit/observed` emits `content_url`, and
  verified `/api/dashboard/media-assets/content?asset_id=...` returns the
  registered QQ image bytes with `image/jpeg`.

Findings:
- The dashboard should remain a thin same-origin browser boundary. It should not
  become the media authorization or filesystem authority; Go still validates the
  registered asset and allowed roots before serving bytes.
- Existing `/api/dashboard/attachments?path=...` remains only for legacy
  workspace upload paths. New Go-registered attachments should use asset ids.

Decision:
- Accepted as an observe-only migration slice. This improves the current QQ
  image/file viewing path without changing message capture or reply behavior.

Follow-ups:
- Carry asset ids into the main session message dashboard, not only shadow
  audit records.
- Add file type previews and OCR/VLM summaries once media download/mirroring is
  fully owned by Go jobs.
