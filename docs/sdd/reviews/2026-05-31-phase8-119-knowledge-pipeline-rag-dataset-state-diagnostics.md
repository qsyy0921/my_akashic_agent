# Review: knowledge pipeline rag dataset state diagnostics

Spec: `docs/sdd/specs/agent-gateway/062-knowledge-pipeline-rag-dataset-state-diagnostics.md`

Implementation summary:
- `KnowledgePipelineView` 新增 `rag_datasets`，按 `dataset_id` 输出 dataset 级 `rag_ingest` stage、checkpoint、checkpoint lag、status 和 reasons。
- 数据来源完全来自 Go-owned `rag_ingest` job payload / metadata 与 `ragflow:*` checkpoint metadata，不调用外部 RAGFlow API。
- `runtime-overview` 新增 `knowledge_pipeline_rag_datasets`、`knowledge_pipeline_rag_dataset_warning`、`knowledge_pipeline_rag_dataset_blocked` summary。

Tests run:
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- `dataset_id` 已经稳定存在于 `rag_ingest` job payload 和 checkpoint metadata，足够支撑 Go control-plane 做 per-dataset drill-down。
- 群级 `rag_ingest` stage 仍然有价值，它是总览；`rag_datasets` 只是把异常精确下钻到具体 dataset。
- 如果 job payload 的 `dataset_id` 与 checkpoint metadata 的 `dataset_id` 不一致，Go 现在会把它们识别成两个 dataset，这符合审计视角，也能暴露错误绑定。

Decision:
- 接受本轮实现。Go 继续只沉淀稳定 dataset 元数据和诊断，不接管外部 RAGFlow index/runtime 行为。

Follow-ups:
- 若后续需要更强索引状态，可继续评估 parse/index 完成态，但前提仍是这些状态先变成 Go-owned control-plane，而不是直接依赖外部 API 即时查询。
