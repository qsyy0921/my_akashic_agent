# Review: knowledge pipeline checkpoint age diagnostics

Spec: `docs/sdd/specs/agent-gateway/060-knowledge-pipeline-checkpoint-age-diagnostics.md`

Implementation summary:
- 在 `KnowledgePipelineCheckpointLagView` 上新增 `age_seconds`，直接复用现有 memory/rag lag 视图，不再增加平行 freshness 结构。
- `KnowledgePipelineDiagnosticsService` 现在基于 checkpoint `updated_at` 和请求 `now` 计算 age，并把 `lag + age >= stale_after_seconds` 归类为 `*_checkpoint_stagnant`。
- `lag danger + stale + stage high pressure` 继续保留更强的 `*_checkpoint_stalled_under_pressure` blocked 语义。
- `runtime-overview` 新增 `knowledge_pipeline_stale_checkpoint_targets` 与 `knowledge_pipeline_stagnant_targets` summary。

Tests run:
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 把 age 直接挂在 lag 视图上就足够表达 control-plane 语义，避免新增一组 `memory_checkpoint_age` / `rag_checkpoint_age` 平行字段。
- `stagnant` 与 `stalled_under_pressure` 需要区分：前者说明“落后且长时间不动”，后者才说明“在高压下已经形成 blocked”。
- stale checkpoint 不必单独提升 pipeline status；若 source 没增长，旧 checkpoint 只是时间老，不代表知识流水线异常。

Decision:
- 接受本轮实现。Go 继续只做只读诊断，不引入调度、副作用或 Python RAG 策略变更。

Follow-ups:
- 后续可继续评估 per-group lease lag 和 dataset/index state，但前提仍是这些状态先成为 Go-owned control-plane。
