# 153. Go Migration Goal Verification Runbook

Date: 2026-06-02

## Status

Accepted

## Context

本地 Go 迁移收口现在依赖多个分散入口：

- `/v1/runtime-config`
- `/v1/runtime-workers`
- `/v1/runtime-overview`
- `/v1/outbound-cutover/readiness`
- `/v1/queue-backend`
- `/v1/receiver-statuses`
- `/v1/knowledge-job-planner/*`
- `scripts/verify-knowledge-planner-cutover.ps1`
- `scripts/verify-telegram-backend.ps1`
- 必要时 `scripts/run-napcat-native-rich-media-smoke.ps1`

这会带来两个问题：

1. 每轮更新 goal 时需要手工拼接多份 live 证据。
2. “QQ rich media / Go outbox owner / Telegram backend / knowledge planner / 其它残留分类” 没有统一的 repo-owned 读取入口。

用户已经明确要求：每轮结束都要真实更新 goal，而不是只复述旧结论。

## Decision

新增 repo-owned 只读脚本：

- `scripts/verify-go-migration-goal.ps1`

该脚本必须：

1. 直接读取当前 live runtime：
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
2. 复用已有 repo-owned 只读验证脚本：
   - `verify-knowledge-planner-cutover.ps1`
   - `verify-telegram-backend.ps1`
3. 输出统一 JSON，至少包含：
   - `qq_outbox_cutover`
   - `qq_rich_media`
   - `telegram_backend`
   - `knowledge_planner`
   - `residual_classification`
   - `checks.open_blockers`
4. 默认保持只读，不发送 QQ/Telegram 消息。
5. 允许通过显式参数执行可选原生 NapCat rich-media probe，用于在需要时刷新当前 rich-media 平台 blocker 证据。
6. 允许通过显式参数执行可选 Go outbox scope live smoke，用于在需要时证明当前已放开的成功路由和仍被 gate 的路由。

## Consequences

正面：

- 后续每轮 goal 更新有固定证据入口。
- Telegram token 缺失、knowledge planner 健康、outbox partial cutover、agent_job external lease 未切换等状态可以一次性读出。
- QQ rich-media 若需要复核，可在同一入口下显式触发原生 probe，而不是临时拼命令。

负面：

- rich-media 的最终平台成功与否仍然不能靠纯只读 endpoint 证明；需要显式 probe 或 live smoke。
- `scheduler / proactive / autoscaling / dashboard fallback` 的分类仍然包含治理性判断，不会自动替代更细的专项验证。

## Verification

最小验证：

1. 运行：

```powershell
.\scripts\verify-go-migration-goal.ps1
```

2. 在需要刷新 QQ rich-media 当前证据时运行：

```powershell
.\scripts\verify-go-migration-goal.ps1 -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
```

3. 在需要同时刷新当前 Go outbox owner 真实边界时运行：

```powershell
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
```

4. 结果必须输出统一 JSON，且清楚区分：
   - Go outbox 当前 cutover scope
   - Telegram backend 是 token 缺失还是 `getMe` 失败
   - knowledge planner 是否仍由 Go 持有 admission owner
   - agent_job external lease / scheduler / proactive / autoscaling / dashboard fallback 当前分类
