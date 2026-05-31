# Review: phase8-116 knowledge pipeline source lag diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/059-knowledge-pipeline-source-lag-diagnostics.md`

Implementation summary:
- `KnowledgePipelineDiagnosticsService` 新增 inbox 依赖，按 observe-only QQ 群读取 bounded inbox sample 中的 `metadata.seq`。
- 新增 `latest_source_seq`、`sequenced_events`、`memory_checkpoint_lag`、`rag_checkpoint_lag_max`。
- runtime overview summary 新增：
  - `knowledge_pipeline_lagging_targets`
  - `knowledge_pipeline_stalled_targets`
- source lag 仍然保持只读，只作为控制面诊断，不触发自动重试、补提 job 或 worker 调度。

Tests run:
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- `metadata.seq` 与 checkpoint cursor 的比较是 Go 侧最稳定的“消息源进度 vs 编排进度”对照，不需要引入 Python 内部 RAG 状态。
- lag 阈值保持保守：`lag>=20` 预警，`lag>=100` 且对应 stage 高压时才认定 stalled，避免日常轻微延迟把 pipeline 误判为阻塞。

Decision:
- 接受当前实现，继续把 source lag 作为只读控制面信号，而不是自动化调度输入。

Follow-ups:
- 后续如需进一步提升，可把 lag 与 per-group checkpoint age、dataset/index metadata 结合，但必须保持 Go/Python 边界清晰。
