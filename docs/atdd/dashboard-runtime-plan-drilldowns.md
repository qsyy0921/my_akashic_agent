# ATDD: Dashboard runtime plan drilldowns

## Scope

- runtime overview dashboard 对以下 Go-owned detail 的结构化只读 drilldown：
  - `agent_job_capacity_plan`
  - `agent_job_priority_plan`
  - `knowledge_job_planner_cutover_plan`
  - `outbound_cutover_plan`

## Preconditions

- Go runtime 在 `http://127.0.0.1:8780` 可访问。
- Python dashboard 在 `http://127.0.0.1:2236` 可访问。
- unified verifier 可执行：
  - `scripts/verify-go-migration-goal.ps1`

## Scenarios

### Scenario 1

- Action:
  请求 `GET /plugins/runtime_overview/dashboard_panel.js`。
- Expect:
  panel 资产中可见：
  - `Agent Job Capacity`
  - `Agent Job Priority`
  - `Knowledge Planner Cutover`
  - `Outbound Cutover Plan`

### Scenario 2

- Action:
  查看 `knowledge_job_planner_cutover_plan` 与 `outbound_cutover_plan` drilldown。
- Expect:
  可直接读到：
  - current / desired / recommended owner
  - `Preview Bucket`
  - `Expected OneBot Channels`
  - `Required Checks / Enable Steps / Verification Steps / Rollback Steps`

### Scenario 3

- Action:
  运行 `.\scripts\verify-go-migration-goal.ps1`。
- Expect:
  `dashboard_read_models` 中：
  - `agent_job_capacity_plan_table=true`
  - `agent_job_priority_plan_table=true`
  - `knowledge_job_planner_cutover_plan_table=true`
  - `outbound_cutover_plan_table=true`
  且 `dashboard_fallback.category=live_verified_runtime_read_models`。

## Failure Signals

- 这些 plan 仍只能从 raw JSON 阅读；
- `Preview Bucket` 或 `Expected OneBot Channels` 不可见；
- verifier 无法反映这些 dashboard read-model 证据。

## Evidence

- `GET http://127.0.0.1:2236/plugins/runtime_overview/dashboard_panel.js`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `.\scripts\verify-go-migration-goal.ps1`
