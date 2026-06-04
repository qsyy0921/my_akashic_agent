# 160 Goal Verifier Scheduler Proactive Live Checks

## Context

- `scripts/verify-go-migration-goal.ps1` 已经是 repo-owned 的统一 goal 更新入口。
- 但其中对剩余 Go 迁移项的 `scheduler`、`proactive`、`dashboard_fallback`、
  `agent_job_external_lease_result_ack` 仍有多处写死为 `not_current_turn`。
- 这会让每轮 goal 更新重新退回“人工解释哪些只是没验证”，而不是基于当前 live
  运行态给出分类。

## Decision

- 把以下 residual 项从静态分类改成当前 turn 的 live 取证：
  - `agent_job_external_lease_result_ack`
  - `scheduler`
  - `proactive`
  - `dashboard_fallback`
- `scheduler` 通过当前 runtime 的：
  - `/v1/scheduler/jobs`
  - `/v1/scheduler/leases`
  - `/v1/scheduler/diagnostics`
  做分类
- `proactive` 通过当前 runtime 的：
  - `/v1/proactive/tick-logs`
  - `/v1/proactive/drift/summary`
  - `/v1/proactive/anyaction/quota`
  - `/v1/proactive/bg-context/main/last`
  - `/v1/proactive/context-only/last`
  以及 dashboard 的：
  - `/api/dashboard/proactive/tick_logs`
  做分类

## Requirements

1. `scripts/verify-go-migration-goal.ps1` 必须输出 `scheduler_runtime` 和
   `proactive_runtime` 结构化 live 证据，而不是只给一句文档性说明。
2. `residual_classification.scheduler.status` 不再允许是
   `not_current_turn`；当前 turn 必须基于 live endpoint 输出：
   - `live_verified`
   - 并区分 `go_control_plane_live_verified_idle` 与
     `go_control_plane_live_verified_with_runtime_state`
3. `residual_classification.proactive.status` 不再允许是
   `not_current_turn`；当前 turn 必须基于 live endpoint 和 dashboard read
   model 输出：
   - `live_verified`
   - 或 `verification_incomplete`
4. `residual_classification.dashboard_fallback` 必须反映当前 dashboard
   proactive tick-log 读路径是否真实可用，而不是固定写成 follow-up。
5. `residual_classification.agent_job_external_lease_result_ack` 必须明确当前
   turn 已真实重跑 readiness/plan，而不是保留“本轮未看”状态。

## Non-Goals

- 不新增 scheduler reminder/recurring 写入或执行逻辑。
- 不新增 proactive AI 语义行为。
- 不改变 QQ/Telegram、knowledge planner、outbox owner 的现有运行态。
