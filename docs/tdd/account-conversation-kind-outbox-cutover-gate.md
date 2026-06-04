# TDD: Account-conversation-kind Outbox Cutover Gate

## Targeted Coverage

1. Outbox lease filtering:
   - account + conversation type overrides account-only and global gates
   - account-only gate remains the fallback when no route gate is present
   - global gate remains the fallback for all other routes
2. Local outbox worker:
   - passes account+conversation-type allowed kinds into `LeaseNext`
3. Runtime config and queue-backend views:
   - expose route-scoped allowed kinds
   - derive `account_conversation_kind_gated` execution scope
4. Manual live verification:
   - first-account private file succeeds automatically through Go local worker
   - first-account group file remains queued
   - second-account group file still succeeds automatically

## Commands

- `C:\\Users\\10495\\AppData\\Local\\Programs\\Go\\bin\\go.exe test ./...`
- `uv run pytest tests/test_agent_gateway_outbox_worker.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Out of Scope

- Automated NapCat / QQ image success tests
- Telegram backend smoke without a real bot token
- Automatic capability discovery from live rich-media outcomes
