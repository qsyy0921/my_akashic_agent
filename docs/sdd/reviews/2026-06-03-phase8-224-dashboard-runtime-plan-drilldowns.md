# Phase 8.224 Review: Dashboard runtime plan drilldowns

## What changed

- `plugins/runtime_overview/dashboard_panel.ts` now renders structured
  read-only drilldown for:
  - `agent_job_capacity_plan`
  - `agent_job_priority_plan`
  - `knowledge_job_planner_cutover_plan`
  - `outbound_cutover_plan`
- `tests/test_runtime_overview_dashboard_plugin.py` now asserts the panel asset
  exposes these labels and fields.
- `scripts/verify-go-migration-goal.ps1` now records current-turn dashboard
  evidence for these plan drilldowns inside `dashboard_read_models`, and upgrades
  `dashboard_fallback` once all related tables are present.

## Live evidence

本轮实际运行：

- `GET http://127.0.0.1:2236/plugins/runtime_overview/dashboard_panel.js`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

关键 live 结果：

- panel 资产当前已真实包含：
  - `Agent Job Capacity`
  - `Agent Job Priority`
  - `Knowledge Planner Cutover`
  - `Outbound Cutover Plan`
  - `Preview Bucket`
  - `Expected OneBot Channels`
- unified goal verifier 当前 `dashboard_read_models` 已返回：
  - `agent_job_capacity_plan_table=true`
  - `agent_job_priority_plan_table=true`
  - `knowledge_job_planner_cutover_plan_table=true`
  - `outbound_cutover_plan_table=true`
- `dashboard_fallback` 当前 turn 结论已升级为：
  - `status=live_verified`
  - `category=live_verified_runtime_read_models`

## Conclusion

这轮把 runtime overview 中一组重要的 plan 类 drilldown 从 raw JSON 收口成了
结构化只读 view。

当前更准确的结论是：

- dashboard 剩余 gap 继续缩小；
- `agent_job external lease`、capacity/priority、knowledge planner cutover、
  outbound cutover 的 operator 可读性都已进入 table-first 状态；
- 这些方向真正还没完成的部分，已回到 cutover/config/live smoke，而不是 UI
  drilldown 本身。
