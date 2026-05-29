# Review: Phase 2.2 Shadow Dashboard Audit

Spec:

- `docs/sdd/specs/agent-architecture/007-shadow-mode-gateway.md`
- `docs/sdd/specs/agent-architecture/004-target-agent-architecture.md`

Implementation summary:

- Add a dashboard plugin named `shadow_audit` instead of changing the main
  dashboard bundle.
- Expose `/api/dashboard/shadow-audit/observed` from Python for read-only
  inspection of shadow-mode events.
- Prefer the Go gateway query endpoint `/v1/shadow/observed` when available.
- Fall back to local `shadow/session.jsonl` and `shadow/inbound.jsonl` so local
  development remains useful when the Go gateway is not running.
- Show route, sender, decision/source, content, metadata, and attachment links
  in the plugin panel.

Safety boundaries:

- The plugin is read-only.
- It does not send messages, modify sessions, or invoke the agent.
- Gateway failures are represented as status metadata and fall back to JSONL in
  `auto` mode.
- Main dashboard assets are not modified in this slice.

Tests run:

- `uv run pytest tests\test_shadow_audit_plugin.py tests\test_shadow_gateway.py tests\test_sdd_contract_fixtures.py -q`
  passed.
- `uv run python -m compileall plugins\shadow_audit tests\test_shadow_audit_plugin.py`
  passed.
- `npx --yes esbuild plugins\shadow_audit\dashboard_panel.ts --outfile=.tmp\shadow_audit_panel_check.js --bundle=false --platform=browser --target=es2020 --format=iife`
  passed.
- `node --check plugins\shadow_audit\dashboard_panel.js` passed.
- `git diff --check -- plugins\shadow_audit tests\test_shadow_audit_plugin.py docs\sdd\reviews\2026-05-30-phase2-2-shadow-dashboard-audit.md docs\sdd\reviews\README.md`
  passed.

Decision:

- Proceed as a visibility slice before moving production routing to Go.

Follow-ups:

- Add persistent Go-side shadow audit storage before relying on gateway-only
  queries across process restarts.
- Add a richer attachment registry when media/file processing moves behind the
  Go gateway.
