# TDD: Account-kind Outbox Cutover Gate

## Targeted Coverage

1. Outbox lease filtering:
   - global allowed kinds remain the fallback
   - account-scoped allowed kinds override the global gate for matching QQ
     account ids
2. Local outbox worker:
   - passes account-scoped allowed kinds into `LeaseNext`
3. Runtime config and queue-backend views:
   - expose account-scoped allowed kinds and the derived
     `account_kind_gated` execution scope
4. Manual live verification:
   - second-account file succeeds automatically through Go local worker
   - first-account file remains queued

## Commands

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...`
- `uv run pytest tests/test_agent_gateway_outbox_worker.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Out of Scope

- Automated NapCat / QQ image success tests
- Telegram backend smoke without a real bot token
- Automatic capability discovery from live rich-media outcomes
