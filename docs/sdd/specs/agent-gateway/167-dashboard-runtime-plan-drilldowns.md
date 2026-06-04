# Spec 167: Dashboard runtime plan drilldowns

## Status

Accepted for the current iteration.

## Context

runtime overview 已经稳定暴露多类 Go-owned 计划与建议类 detail：

1. `agent_job_capacity_plan`
2. `agent_job_priority_plan`
3. `knowledge_job_planner_cutover_plan`
4. `outbound_cutover_plan`

这些 detail 已经是结构化 JSON，但 dashboard runtime overview panel 仍把它们主要留在
raw JSON 区域。operator 虽然能从 payload 里读到：

- owner / cutover 决策
- verification steps
- enable / rollback steps
- capacity / priority recommendation

但当前 UI 仍缺少稳定的只读 drilldown。

## Decision

本轮把以上四类 detail 提升为结构化只读 drilldown：

1. `agent_job_capacity_plan`
   - 展示 summary KPI；
   - 展示按 job type 的 capacity table；
   - 展示 verification steps。
2. `agent_job_priority_plan`
   - 展示 summary KPI；
   - 展示按 rank/job type 的 priority table；
   - 展示 verification steps。
3. `knowledge_job_planner_cutover_plan`
   - 展示 current / desired / recommended admission owner；
   - 展示 readiness / preview bucket / preview jobs；
   - 展示 required / enable / verification / rollback steps。
4. `outbound_cutover_plan`
   - 展示 current / desired / recommended execution owner；
   - 展示 readiness / smoke cases / expected OneBot channels；
   - 展示 required / enable / verification / rollback steps。

全部继续保持只读：

- 不修改 env；
- 不启动 worker；
- 不发送 QQ/Telegram；
- 不 ack/nack MQ；
- 不创建或执行 AgentJob；
- 不触发 Python AI。

## Out of Scope

- 启用 outbound / agent_job 真正 cutover；
- 恢复 QQ 群发；
- 处理 Telegram token 缺失；
- 解决 QQ image native blocker；
- 落地 autoscaling / priority executor。

## Acceptance

- `GET /plugins/runtime_overview/dashboard_panel.js` 可见：
  - `Agent Job Capacity`
  - `Agent Job Priority`
  - `Knowledge Planner Cutover`
  - `Outbound Cutover Plan`
  - `Expected OneBot Channels`
  - `Preview Bucket`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` 通过。
- `.\scripts\verify-go-migration-goal.ps1` 的 `dashboard_read_models` 返回：
  - `agent_job_capacity_plan_table=true`
  - `agent_job_priority_plan_table=true`
  - `knowledge_job_planner_cutover_plan_table=true`
  - `outbound_cutover_plan_table=true`
- `dashboard_fallback.category=live_verified_runtime_read_models`。
