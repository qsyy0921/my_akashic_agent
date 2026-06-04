# SDD: Dashboard Proactive Tick Logs Runtime Fallback Live Verifier

## Problem

Go 已拥有 proactive tick-log 的确定性状态与 HTTP read-model，但 dashboard
`/api/dashboard/proactive/tick_logs` 的 runtime fallback 之前只在 pytest
级 contract 中覆盖，缺少 repo-owned isolated live verifier。

需要一个当前 turn 可重复执行的 verifier，证明在 SQLite `tick_log` mirror
为空时，dashboard 仍能从 Go runtime 读取 tick log list/detail/steps，且不把
runtime 数据回写进 SQLite fallback。

## Non-goals

- 不新增 proactive 业务策略、AI 推理、tool execution 或 LLM 调用。
- 不修改 dashboard tick-log 的 operator UX。
- 不把 SQLite mirror 删除逻辑迁入 Go。

## Requirements

1. verifier 必须启动隔离 temp `agent-runtime`。
2. verifier 必须通过 Go `/v1/proactive/tick-logs/start|finish` 和
   `/v1/proactive/tick-steps` 写入至少一条 runtime tick。
3. verifier 必须在独立 dashboard workspace 中保持 SQLite `tick_log`
   为空。
4. verifier 必须通过 dashboard `/api/dashboard/proactive/tick_logs`、
   `/api/dashboard/proactive/tick_logs/{tick_id}`、
   `/api/dashboard/proactive/tick_logs/{tick_id}/steps` 证明 fallback 可读。
5. verifier 必须证明 dashboard 读取后 SQLite `tick_log` 仍为空。
6. unified goal verifier 必须收集这条 current-turn evidence。

## Invariants

- side effect 只允许：
  - temp runtime state 写入；
  - temp dashboard workspace SQLite 初始化。
- 不发送 QQ / Telegram。
- 不启动 Python AI loop。
- 不把 runtime tick-log 数据镜像回 SQLite fallback。

## Acceptance

- `scripts/verify-dashboard-proactive-tick-logs-boundary.ps1` 返回
  `conclusion.status=live_verified`
- `conclusion.category=dashboard_proactive_tick_logs_runtime_fallback_live_verified`
- `scripts/verify-go-migration-goal.ps1` 输出中包含
  `dashboard_proactive_tick_logs_boundary`
  和对应 residual classification。
