# Review: dual-account rich media split verification

Spec:
- `docs/sdd/specs/agent-gateway/147-dual-account-rich-media-split-verification.md`

Implementation summary:
- Queried the second QQ account's group list from NapCat.
- Ran native rich-media smoke on second-account groups.
- Ran Akashic manual file/image delivery-dispatch on second-account group
  `284331268`.
- Updated SDD state docs with account/media-kind split conclusions.

Tests run:
- second-account `get_group_list` on `ws://127.0.0.1:3002`
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -GroupId 869134328`
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -GroupId 284331268`
- Akashic manual second-account file dispatch to `284331268`
- Akashic manual second-account image dispatch to `284331268`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Second account `2365524513` has accessible QQ groups, so cross-account
  verification is possible.
- On the second account, native file send succeeds while native image send
  still fails with `rich media transfer failed`.
- Akashic manual delivery-dispatch matches native second-account behavior:
  file send succeeds, image send fails with the same platform error.
- Therefore the blocker is now split:
  - image send is a cross-account native/platform issue, not an Akashic issue;
  - first-account file failure is account-session-specific;
  - global Go outbox expansion beyond `text_only` remains unsafe because the
    runtime lacks per-account capability gates.

Decision:
- Keep goal active.
- Keep global Go execution owner at `text_only`.
- Treat next meaningful QQ action as first-account re-login/session replacement,
  while image send still requires broader platform-side investigation.
