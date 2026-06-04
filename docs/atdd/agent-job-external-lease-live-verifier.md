# Agent Job External Lease Live Verifier ATDD

## Scenario

本地 runtime 仍处于 state-store ownership，operator 需要一个 repo-owned
脚本明确当前 `agent_job external lease result-ack` 到底卡在什么地方。

## Acceptance

1. `.\scripts\verify-agent-job-external-lease.ps1` 返回结构化 JSON。
2. 输出包含：
   - `readiness`
   - `plan`
   - `runtime_flags`
   - `queue_backend`
   - `queue_topology`
   - `runtime_overview`
   - `blocker_buckets`
   - `conclusion`
3. 当前本地运行态下：
   - `conclusion.status=live_verified`
   - `conclusion.current_scope=state_store_lease`
   - `conclusion.category=blocked_by_configuration_and_cutover_flags`
4. `blocker_buckets.configuration` 非空。
5. `checks.worker_coverage_ready=true`，避免把当前 blocker 误记成 worker coverage。
6. `scripts/verify-go-migration-goal.ps1` 输出中新增 `agent_job_external_lease`
   段，并让 `residual_classification.agent_job_external_lease_result_ack`
   复用同一结论。

## Failure Signals

- verifier 仍需要人工读多个 endpoint 才能解释 blocker
- unified goal verifier 仍只给几行摘要
- 当前 state-store blocker 被误记成 worker coverage 或未知原因
