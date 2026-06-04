# Goal Verifier Scheduler Proactive Live Checks ATDD

## Scenario

统一 goal verifier 在当前本地 runtime 上运行，需要把 scheduler、
proactive 和 dashboard fallback 的残留分类建立在真实 endpoint 证据上。

## Acceptance

1. `.\scripts\verify-go-migration-goal.ps1` 输出 `scheduler_runtime`。
2. `scheduler_runtime.checks` 至少包含：
   - `jobs_endpoint_reachable=true`
   - `leases_endpoint_reachable=true`
   - `diagnostics_endpoint_reachable=true`
3. `residual_classification.scheduler.status=live_verified`，不再是
   `not_current_turn`。
4. `.\scripts\verify-go-migration-goal.ps1` 输出 `proactive_runtime`。
5. `proactive_runtime.checks` 至少包含：
   - `recent_tick_logs_present=true`
   - `anyaction_quota_visible=true`
   - `dashboard_tick_logs_readable=true`
6. `residual_classification.proactive.status=live_verified`，不再是
   `not_current_turn`。
7. `residual_classification.dashboard_fallback.status=live_verified`。
8. `residual_classification.agent_job_external_lease_result_ack.status=live_verified`。

## Failure Signals

- unified verifier 仍输出 `scheduler.status=not_current_turn`
- unified verifier 仍输出 `proactive.status=not_current_turn`
- dashboard fallback 仍只给静态 follow-up 文案而没有当前 turn 证据
