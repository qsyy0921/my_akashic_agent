# 201 Worker Control Executor Boundary Live Verifier

## Context

- Go 已经拥有：
  - `agent_job capacity plan`
  - `agent_job priority plan`
  - `operator approval`
  - `control mutation audit`
  - `control mutation policy`
- 但当前剩余问题不是“看不到 control plane”，而是：
  - runtime 里还没有真正的 autoscaling / concurrency / priority executor
  - unified goal verifier 之前只用写死文案描述这件事
- 本切片要把这条剩余项从静态说明提升成 repo-owned live verifier。

## Decision

新增 repo-owned verifier：

```text
scripts/verify_worker_control_executor_boundary.py
scripts/verify-worker-control-executor-boundary.ps1
```

该 verifier 必须只读校验：

1. `/v1/agent-job-capacity/plan`
2. `/v1/agent-job-priority/plan`
3. `/v1/control-mutations/policy`
4. `/v1/runtime-workers`
5. `/v1/runtime-overview.summary`

并给出当前 turn 结论：Go 是否已经具备 worker-control control plane，
以及 runtime 中是否已经出现真实 executor。

## Requirements

1. verifier 输出至少包含：
   - `runtime_overview_summary`
   - `capacity_plan`
   - `priority_plan`
   - `control_mutation_policy`
   - `runtime_workers`
   - `checks`
   - `conclusion`
2. `checks` 至少包含：
   - capacity / priority plan 与 runtime-overview summary 的 ready/reason/blockers
     一致性
   - priority plan 仍为 `manual_only`
   - priority plan 的 `worker_control_owner` 仍为 `python`
   - control mutation policy 已包含：
     - `agent_job_capacity: apply, rollback`
     - `agent_job_priority: apply, rollback`
   - runtime workers 中当前没有 autoscaling / concurrency / priority executor
     candidate
3. `conclusion.category` 至少区分：
   - `go_control_plane_live_verified_without_worker_control_executor`
   - `go_worker_control_executor_present`
   - `worker_control_executor_boundary_mismatch_detected`
4. `scripts/verify-go-migration-goal.ps1` 必须接入该 verifier，并输出：
   - `worker_control_executors`
   - `residual_classification.worker_control_executors`
5. 为兼容现有消费者，`autoscaling_executor` 可以继续保留，但必须复用同一 live
   结论，而不能继续使用写死文案。

## Non-Goals

- 不自动启动或缩容 Python worker
- 不自动修改并发、优先级或配置
- 不 apply / rollback control mutation
- 不把 plan-only 状态误报成 executor 已落地
- 不迁移 Python 的模型、RAG、OCR/VLM、tool execution 或 reasoning
