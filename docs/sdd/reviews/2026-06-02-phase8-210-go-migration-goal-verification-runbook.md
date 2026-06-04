# Phase 8.210 Review - Go Migration Goal Verification Runbook

Date: 2026-06-02

## Summary

新增 repo-owned 统一 goal 验证脚本 `scripts/verify-go-migration-goal.ps1`，把当前 Go 迁移收口最关键的四类结论拉进单个 JSON：

- QQ outbox cutover scope
- QQ rich-media 当前 blocker
- Telegram backend token / receiver / `getMe` 状态
- knowledge planner 当前健康状态

同时补充 `agent_job external lease / scheduler / proactive / autoscaling / dashboard fallback / Python-owned AI runtime` 的状态分类，减少后续每轮 goal 更新对手工拼接多份 live 证据的依赖。

## What Changed

- 新增 `scripts/verify-go-migration-goal.ps1`
- 复用已有：
  - `scripts/verify-knowledge-planner-cutover.ps1`
  - `scripts/verify-telegram-backend.ps1`
  - 可选 `scripts/run-napcat-native-rich-media-smoke.ps1`
- 用 live endpoint 聚合：
  - `/healthz`
  - `/v1/runtime-config`
  - `/v1/runtime-workers`
  - `/v1/runtime-overview`
  - `/v1/outbound-cutover/readiness`
  - `/v1/queue-backend`
  - `/v1/receiver-statuses`
  - `/v1/jobs?type=group_memory_extract`
  - `/v1/agent-job-external-lease/readiness`
  - `/v1/agent-job-external-lease/plan`
  - `/v1/agent-job-capacity/plan`
  - `/v1/agent-job-priority/plan`

## Verification

已运行：

```powershell
.\scripts\verify-go-migration-goal.ps1 -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

本轮 live 结果应体现：

- Go outbox 当前仍为 `account_conversation_kind_gated`
- Telegram 当前仍为 `token_missing`
- knowledge planner 当前仍健康，且 Go 持有 admission owner
- 第一账号 `3219982` group rich-media native probe 仍返回 `rich media transfer failed`

## Risks / Follow-up

- 该脚本统一了 goal 验证入口，但不会替代真实 QQ/Telegram smoke。
- rich-media 平台 blocker 若发生变化，仍需要显式 probe 或 live smoke 刷新证据。
- `scheduler / proactive / autoscaling / dashboard fallback` 目前仍是分类性结论，不是专项完成证明。
