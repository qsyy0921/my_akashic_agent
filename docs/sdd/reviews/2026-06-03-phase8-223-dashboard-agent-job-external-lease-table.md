# Phase 8.223 Review: Dashboard agent_job external lease table

## What changed

- `plugins/runtime_overview/dashboard_panel.ts` now renders structured
  read-only drilldown for:
  - `agent_job_external_lease_readiness`
  - `agent_job_external_lease_plan`
- The panel now exposes stable sections for:
  - current / desired / recommended owner
  - worker coverage
  - blockers / notes
  - required / enable / verification / rollback steps
- `tests/test_runtime_overview_dashboard_plugin.py` now asserts that the panel
  asset exposes these external-lease read-model labels.
- `scripts/verify-go-migration-goal.ps1` now includes dashboard-panel evidence
  for the external-lease drilldown when classifying `dashboard_fallback`.

## Live evidence

本轮实际运行：

- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `GET http://127.0.0.1:2236/plugins/runtime_overview/dashboard_panel.js`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

关键 live 结果：

- dashboard panel asset 当前已真实包含：
  - `Agent Job External Lease`
  - `Agent Job External Lease Plan`
  - `Current Owner`
  - `Desired Owner`
  - `Recommended Owner`
  - `Required Checks`
  - `Enable Steps`
  - `Verification Steps`
  - `Rollback Steps`
- unified goal verifier 当前已把 `dashboard_fallback` 的当前 turn 证据扩大到：
  - proactive dashboard tick-log read-model
  - `agent_job external lease` structured drilldown read-model

## Conclusion

这轮把 runtime overview 中 `agent_job external lease` 的 dashboard
read-model gap 收掉了。

当前更准确的结论是：

- `agent_job external lease result-ack` 真正未完成的部分仍是
  NATS/config/cutover flags，不是 dashboard 可读性；
- dashboard 这条线又少了一块 raw-JSON-only drilldown；
- 剩余 blocker 仍是 Telegram token、QQ image native blocker，以及
  `agent_job external lease` 的真实 external cutover 条件。
