# TDD: Native Private Rich-media Route Verification

## Targeted Coverage

1. Native NapCat rich-media smoke script:
   - supports `ConversationType=private`
   - switches image send to `send_private_msg`
   - switches file send to `upload_private_file`
2. Manual live verification:
   - native private image/file behavior on both QQ accounts
   - Akashic private image/file behavior on the first QQ account

## Commands

- `scripts/run-napcat-native-rich-media-smoke.ps1 -ConversationType private -ChatId 2365524513`
- `scripts/run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -ConversationType private -ChatId 1049511700`
- manual `POST /v1/outbound` + `POST /v1/delivery-dispatch/send` verification for first-account private image/file
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Out of Scope

- Automated PowerShell unit tests for the smoke script
- Telegram backend smoke without a real bot token
- Automatic widening of Go outbox ownership from this evidence alone
