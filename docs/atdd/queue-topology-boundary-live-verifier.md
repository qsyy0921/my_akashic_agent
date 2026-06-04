# Queue Topology Boundary Live Verifier ATDD

## Scenario

operator 需要一个 repo-owned 入口，在不改动当前长期运行 runtime 的前提下，
直接证明 queue topology、queue backend 和 runtime-overview queue summary
对 execution boundary 的描述是一致的。

## Acceptance

1. `.\scripts\verify-queue-topology-boundary.ps1` 返回结构化 JSON。
2. 输出包含：
   - `queue_backend`
   - `queue_topology`
   - `runtime_overview_summary`
   - `checks`
   - `conclusion`
3. 当前本机运行后：
   - `checks.provider_matches_queue_backend=true`
   - `checks.outbox_execution_owner_matches_queue_backend=true`
   - `checks.agent_job_execution_owner_matches_queue_backend=true`
   - `checks.agent_job_ack_owner_matches_runtime_overview_summary=true`
   - `checks.external_lease_ready_matches_runtime_overview_summary=true`
   - `conclusion.status=live_verified`
   - `conclusion.category=queue_topology_boundary_live_verified_with_runtime_overview`
4. `scripts/verify-go-migration-goal.ps1` 输出中新增 `queue_topology` 段，并让
   `residual_classification.queue_topology_boundary` 复用同一结论。
5. unified verifier 当前 turn 还必须返回
   `dashboard_read_models.queue_topology_table=true`。

## Failure Signals

- 仍需要人工比对三个 endpoint 才能知道 owner/ack-owner 是否一致
- unified goal verifier 看不到 queue-topology live 证据
- panel `Queue Topology` 实际文案与静态资产检查不一致，导致当前 turn 误报
