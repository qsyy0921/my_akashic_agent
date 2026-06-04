# Review: native napcat rich media comparison

Spec:
- `docs/sdd/specs/agent-gateway/143-native-napcat-rich-media-comparison.md`

Implementation summary:
- Added `scripts/run-napcat-native-rich-media-smoke.ps1` to replay the same
  streamed `upload_file_stream` protocol as the Go OneBot adapter directly
  against NapCat.
- Re-ran native image/file group send against `ws://127.0.0.1:3001` and group
  `27234224`.
- Re-verified runtime/dashboard state for outbound cutover, Telegram token, and
  knowledge planner persistence.

Tests run:
- `.\scripts\run-napcat-native-rich-media-smoke.ps1`
- `Invoke-RestMethod http://127.0.0.1:8780/v1/outbound-cutover/readiness`
- `Invoke-RestMethod http://127.0.0.1:8780/v1/knowledge-job-planner/cutover-plan`
- `Invoke-RestMethod http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Native NapCat image/file upload now reaches `file_complete` with `file_path`
  and then fails at final QQ rich-media send with the same
  `rich media transfer failed` error seen through Akashic.
- Current local outbox worker is still a global enable/disable cutover, not a
  per-kind capability gate, so text-only default cutover is not yet a stable
  repo-supported mode.
- Dashboard/runtime overview still shows Telegram token missing and knowledge
  planner current owner as `go_runtime_knowledge_job_planner`.

Decision:
- Treat current QQ rich-media failure as a NapCat/QQ session blocker, not a Go
  adapter blocker.
- Keep default outbox execution owner unchanged until either rich media works
  or the runtime gains a supported capability gate.

Follow-ups:
- Re-login or replace the current NapCat/QQ rich-media session and rerun the
  native comparison plus `run-qq-group-live-smoke.ps1`.
- Only revisit default Go outbox cutover after native rich-media parity passes
  or after a deliberate per-kind execution-owner design lands.
