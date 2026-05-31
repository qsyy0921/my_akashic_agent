# Review: phase8-114 agent job worker coverage diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/057-agent-job-worker-coverage-diagnostics.md`

Implementation summary:
- 在 Go `runtime-overview` 聚合层新增 `AgentJobWorkerCoverageView`，把 Go-owned `AgentJob` pressure 与 Python `agent-worker-statuses` 关联。
- 新增 `job_type -> expected worker_type[]` 只读映射，首批覆盖 `group_memory_extract`、`rag_ingest`、`rag_eval`、`image_generation`。
- `/v1/runtime-overview` 新增：
  - top-level `agent_job_worker_coverage`
  - summary 字段 `agent_job_worker_coverage_*`
  - `Agent Job Worker Coverage` card

Tests run:
- `go test ./app/service -run "TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 这层诊断适合留在 Go aggregate，而不应把 `AgentWorkerStatuses` 直接耦合进 `/v1/job-metrics`，否则 `AgentJobMetricsService` 会从纯 lifecycle/metrics 聚合退化成跨子系统组合器。
- `high pressure + no active worker` 应明确打成 `danger`；`failed/stale/no active` 但非 high pressure 维持 `warn`，避免误把“暂时空闲但未积压”的状态升级成错误。

Decision:
- 接受当前实现，保持只读，不引入 autoscaling、优先级调度或 worker 自恢复副作用。

Follow-ups:
- 如果后续做 worker concurrency/autoscaling，优先基于 `AgentJob` pressure + worker coverage 两者共同判断。
- 如果后续新增 Python worker 类型，需要同步扩展 `job_type -> worker_type` 映射并补 service test。
