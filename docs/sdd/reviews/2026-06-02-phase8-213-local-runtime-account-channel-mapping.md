# Review: local runtime account channel mapping

Spec:
- `docs/sdd/specs/agent-gateway/156-local-runtime-account-channel-mapping.md`

Implementation summary:
- Added default `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` export to the repo-local
  runtime launcher.
- Restarted the local runtime with the same gated Go outbox settings.
- Re-ran repo-owned outbox scope smoke, native NapCat parity, and unified goal
  verification.

Tests run:
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -GroupId 284331268`
- `.\scripts\verify-go-outbox-scope-live.ps1`
- `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- The local launcher had been missing per-account channel mapping, which could
  route second-account outbox work through the primary `qq` alias.
- After adding the mapping and restarting the runtime, second-account group file
  outbox smoke returned to `succeeded`.
- Native NapCat parity still shows first-account group rich-media blocker and
  cross-route image blocker, so the overall goal remains active.

Decision:
- Accept.

Follow-ups:
- Keep the route-scoped Go owner boundary.
- Continue treating Telegram token absence and first-account/image rich-media
  failures as open blockers.
