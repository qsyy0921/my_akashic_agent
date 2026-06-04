# 200 Queue Topology Boundary Live Verifier

## Context

- `queue_topology` read model、runtime overview `queue_topology_*` summary 和
  dashboard `Queue Topology` drilldown 都已经存在。
- 但在本轮之前，还缺一个 repo-owned、可重复执行的 live verifier，直接证明：
  - `/v1/queue-topology`
  - `/v1/queue-backend`
  - `/v1/runtime-overview.summary`
  对 provider recommendation、execution owner、ack owner 和
  external-lease-ready 的描述是一致的。
- 当前主线剩余问题已经从“看不到 queue topology”收敛到“什么时候真正做
  external-lease / result-ack cutover”，因此 verifier 需要把可见性和边界一致性
  固定成当前 turn 证据。

## Decision

新增 repo-owned verifier：

```text
scripts/verify_queue_topology_boundary.py
scripts/verify-queue-topology-boundary.ps1
```

该 verifier 必须：

1. 只读请求 `/v1/queue-backend`、`/v1/queue-topology`、`/v1/runtime-overview`；
2. 校验以下一致性：
   - `provider`
   - `selected_provider`
   - `recommended_provider` vs `recommended_first_backend`
   - `outbox_delivery.execution_owner`
   - `agent_job.execution_owner`
   - `agent_job.ack_owner`
   - `external_lease_ready`
3. 输出结构化 JSON，而不是要求人工比对多个 endpoint；
4. 接入 `scripts/verify-go-migration-goal.ps1`，形成当前 turn 的统一取证；
5. 让 dashboard `Queue Topology` 静态资产检查与真实 panel 空态文案保持一致。

## Requirements

1. verifier 输出至少包含：
   - `queue_backend`
   - `queue_topology`
   - `runtime_overview_summary`
   - `checks`
   - `conclusion`
2. `checks` 至少包含：
   - `provider_matches_queue_backend`
   - `recommended_provider_matches_queue_backend`
   - `outbox_work_kind_visible`
   - `agent_job_work_kind_visible`
   - `outbox_execution_owner_matches_runtime_overview_summary`
   - `agent_job_execution_owner_matches_runtime_overview_summary`
   - `agent_job_ack_owner_matches_runtime_overview_summary`
   - `external_lease_ready_matches_runtime_overview_summary`
3. `conclusion.category` 至少区分：
   - `queue_topology_boundary_live_verified_with_runtime_overview`
   - `queue_topology_boundary_mismatch_detected`
4. `scripts/verify-go-migration-goal.ps1` 必须：
   - 输出 `queue_topology`
   - 输出 `residual_classification.queue_topology_boundary`
   - 输出 `dashboard_read_models.queue_topology_table`
5. verifier 和 unified verifier 都必须保持只读：
   - 不 publish/lease/ack/nack/term MQ
   - 不创建 AgentJob / outbox delivery
   - 不触发 Python AI

## Non-Goals

- 不切换 `queue provider`
- 不启用 NATS external lease
- 不改变 `agent_job` 或 outbox execution owner
- 不把 dashboard 只读证据误表述成 production cutover 完成
