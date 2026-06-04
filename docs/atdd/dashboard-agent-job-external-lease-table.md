# ATDD: Dashboard agent_job external lease table

## Scope

- runtime overview dashboard 对：
  - `agent_job_external_lease_readiness`
  - `agent_job_external_lease_plan`
  的结构化只读 drilldown。

## Preconditions

- Go runtime 在 `http://127.0.0.1:8780` 可访问。
- Python dashboard 在 `http://127.0.0.1:2236` 可访问。
- repo 内 unified verifier 可执行：
  - `scripts/verify-go-migration-goal.ps1`

## Scenarios

### Scenario 1

- Action:
  请求 `GET /api/dashboard/runtime-overview`，并打开 runtime overview panel 资产。
- Expect:
  panel 资产中不再只有 raw JSON；可见：
  - `Agent Job External Lease`
  - `Agent Job External Lease Plan`
  - `Current Owner`
  - `Desired Owner`
  - `Recommended Owner`

### Scenario 2

- Action:
  查看 `agent_job_external_lease_plan` drilldown。
- Expect:
  可读到：
  - `Required Checks`
  - `Enable Steps`
  - `Verification Steps`
  - `Rollback Steps`
  且这些内容来自 runtime overview detail，不触发 mutation。

### Scenario 3

- Action:
  运行 `.\scripts\verify-go-migration-goal.ps1`。
- Expect:
  `dashboard_fallback` 当前 turn 结论不再只依赖 proactive tick logs；
  也包含 `agent_job external lease` structured read-model 的证据。

## Failure Signals

- dashboard panel 里仍只能看到 raw JSON；
- `agent_job_external_lease_plan` 没有 steps table；
- `Current Owner / Desired Owner / Recommended Owner` 不可见；
- unified verifier 仍无法反映这次新增的 read-model 证据。

## Evidence

- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `GET http://127.0.0.1:2236/plugins/runtime_overview/dashboard_panel.js`
- `.\scripts\verify-go-migration-goal.ps1`
