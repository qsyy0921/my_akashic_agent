# Go Outbox Scope Live Verification TDD

Date: 2026-06-02

## Scope

保护当前 Go default owner 的 live smoke 入口与统一 goal 验证集成。

## Target Code Paths

- `scripts/verify-go-outbox-scope-live.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| allowed private file | live smoke | first-account private file auto succeeds |
| allowed second-account group file | live smoke | second-account group file auto succeeds |
| gated first-account group file | live smoke | event stays queued with zero attempts |
| gated first-account private image | live smoke | event stays queued with zero attempts |
| unified goal integration | live smoke | combined verifier embeds outbox scope smoke only when explicitly requested |

## Required Automated Tests

```powershell
.\scripts\verify-go-outbox-scope-live.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

## Deferred Coverage

- QQ rich-media 平台成功与否仍由原生 NapCat probe 和现有 live smoke 负责。
- 该脚本不会证明 Telegram backend 或 knowledge planner，只证明 outbox scope。
