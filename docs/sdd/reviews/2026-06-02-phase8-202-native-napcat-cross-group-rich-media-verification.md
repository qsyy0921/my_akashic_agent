# Review: native napcat cross-group rich media verification

Spec:
- `docs/sdd/specs/agent-gateway/145-native-napcat-cross-group-rich-media-verification.md`

Implementation summary:
- Re-verified current runtime state for text-only Go outbox owner, Telegram
  token presence, and knowledge planner ownership.
- Re-ran the native NapCat rich-media comparison on two additional
  observe-only QQ groups: `3219982` and `164369633`.
- Updated SDD status docs so the remaining blocker is described as a current
  NapCat / QQ session capability issue rather than a single-group anomaly.

Tests run:
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 3219982`
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 164369633`
- `Invoke-WebRequest http://127.0.0.1:8780/healthz`
- `Invoke-WebRequest http://127.0.0.1:8780/v1/runtime-config`
- `Invoke-WebRequest http://127.0.0.1:8780/v1/outbound-cutover/readiness`
- `Invoke-WebRequest http://127.0.0.1:8780/v1/knowledge-job-planner/cutover-plan`
- `Invoke-WebRequest http://127.0.0.1:8780/v1/receiver-statuses`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- On both additional groups, `upload_file_stream` staging still succeeds for
  image and file, but final `send_group_msg` and `upload_group_file` fail with
  the same `rich media transfer failed` platform error.
- Combined with the earlier `27234224` parity result, the blocker is now best
  described as session-wide evidence for the current NapCat / QQ rich-media
  session.
- Text-only Go outbox owner remains healthy and narrow; Telegram backend still
  lacks bot token; knowledge planner remains owned by Go.

Decision:
- Keep goal active.
- Keep QQ rich-media classified as a current session/platform blocker until a
  fresh NapCat / QQ session disproves it.
- Keep Go default execution owner narrow to `text_only`.

Follow-ups:
- Re-login or replace the current NapCat / QQ rich-media session, then rerun
  native comparison and Akashic group smoke.
- Add Telegram bot token and perform backend smoke separately.
