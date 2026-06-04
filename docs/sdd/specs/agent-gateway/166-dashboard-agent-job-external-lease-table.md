# Spec 166: Dashboard agent_job external lease table

## Status

Accepted for the current iteration.

## Context

Go runtime 已经把以下 detail 稳定聚合进 `/v1/runtime-overview` 与
dashboard `/api/dashboard/runtime-overview`：

1. `agent_job_external_lease_readiness`
2. `agent_job_external_lease_plan`

它们已经是结构化数据，但 runtime overview panel 仍然只把这两项作为 raw JSON
显示。operator 虽然能从 payload 里读到：

- current / desired / recommended owner
- queue provider / mode
- worker coverage
- blockers
- required / enable / verification / rollback steps

但当前 dashboard 还没有把这些字段变成稳定的只读表格 drilldown。

## Decision

本轮给 runtime overview panel 增加只读结构化渲染：

1. `agent_job_external_lease_readiness`：
   - 展示 readiness KPI；
   - 展示 worker coverage table；
   - 展示 blockers 和 notes。
2. `agent_job_external_lease_plan`：
   - 展示 current / desired / recommended owner、queue provider / mode；
   - 展示 required checks / enable steps / verification steps / rollback steps；
   - 展示 blockers 和 notes。
3. 保留 raw JSON fallback，但不再把这两类 detail 只留给人工读 payload。
4. 该 panel 继续保持只读：
   - 不修改 env；
   - 不执行 cutover；
   - 不 ack/nack MQ；
   - 不创建或执行 AgentJob；
   - 不触发 Python AI。

## Out of Scope

- 启用 `agent_job` external lease result-ack；
- 变更 MQ provider；
- 启动 Python worker；
- 执行 control mutation；
- 恢复 QQ 群发；
- 处理 Telegram token 缺失。

## Acceptance

- `GET /api/dashboard/runtime-overview` 仍返回：
  - `agent_job_external_lease_readiness`
  - `agent_job_external_lease_plan`
- `GET /plugins/runtime_overview/dashboard_panel.js` 可见以下结构化 drilldown 文本：
  - `Agent Job External Lease`
  - `Agent Job External Lease Plan`
  - `Current Owner`
  - `Desired Owner`
  - `Recommended Owner`
  - `Required Checks`
  - `Enable Steps`
  - `Verification Steps`
  - `Rollback Steps`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"` 通过。
- `.\scripts\verify-go-migration-goal.ps1` 的 `dashboard_fallback` 当前 turn 结论反映：
  - proactive dashboard read-model 可见；
  - `agent_job external lease` structured drilldown 也可见。
