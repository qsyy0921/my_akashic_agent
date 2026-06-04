# Goal Verification Hardening TDD

Date: 2026-06-02

## Scope

保护三类修复：

1. OneBot WebSocket 响应 `status` 字段容错
2. 本地 launcher 显式启用 outbox worker
3. Telegram verifier 的配置来源证据

## Target Code Paths

- `services/agent-runtime/infrastructure/onebotdelivery/adapter.go`
- `services/agent-runtime/infrastructure/onebotdelivery/adapter_test.go`
- `scripts/start-agent-runtime.ps1`
- `scripts/verify-telegram-backend.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| structured status field | go test | non-string `status` does not prevent platform error classification |
| launcher worker flag | live smoke | outbox worker starts with explicit switch |
| telegram config evidence | script | verifier reports channel declared but token placeholder unresolved |

## Required Automated Tests

```powershell
C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./infrastructure/onebotdelivery
.\scripts\verify-telegram-backend.ps1 -RepoRoot E:\agent\my-akashic_agent
.\scripts\verify-go-outbox-scope-live.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

## Deferred Coverage

- Telegram 实际 `getMe` / receiver / 收发 smoke 仍取决于外部 token。
- QQ image / first-account group-file 平台 blocker 仍取决于外部会话状态。
