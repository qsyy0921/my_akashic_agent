# Review: knowledge pipeline job lease freshness diagnostics

Spec: `docs/sdd/specs/agent-gateway/061-knowledge-pipeline-job-lease-freshness-diagnostics.md`

Implementation summary:
- `KnowledgePipelineJobStageView` 新增 pending/active age、stale/expired active lease 和 freshness status/reason。
- `KnowledgePipelineDiagnosticsService` 现在按群聚合 `group_memory_extract` / `rag_ingest` 的 stage freshness，并把 `*_lease_stale` / `*_lease_expired` 直接映射到 pipeline warn/blocked。
- `runtime-overview` 新增 `knowledge_pipeline_expired_active_lease_targets` 与 `knowledge_pipeline_stale_active_lease_targets` summary。

Tests run:
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 把 lease freshness 放进现有 stage 视图，比新建 pipeline-level lease section 更合适，因为它天然属于 `group_memory` / `rag_ingest` 的执行态。
- `expired_active_lease` 和 checkpoint lag/stagnant 需要并存：前者说明执行态卡住，后者说明知识落后；二者不能互相替代。
- `old_pending_backlog` 只提升到 warn，不直接 blocked；真正 blocked 的执行态信号仍然是 expired active lease。

Decision:
- 接受本轮实现。Go 继续只扩展确定性 control-plane 可观察性，不改变 Python worker 执行协议。

Follow-ups:
- 后续若继续推进 knowledge control-plane，可考虑 dataset/index state，但前提仍是这些状态先成为 Go-owned。
