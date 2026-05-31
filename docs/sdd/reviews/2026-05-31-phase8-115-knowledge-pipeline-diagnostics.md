# Review: phase8-115 knowledge pipeline diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/058-knowledge-pipeline-diagnostics.md`

Implementation summary:
- 新增 Go `KnowledgePipelineDiagnosticsService`，按 observe-only QQ 群聚合：
  - observe capture readiness
  - `group_memory_extract` / `rag_ingest` group-level backlog
  - memory / ragflow checkpoints
  - Python worker coverage
- 新增 `GET /v1/knowledge-pipeline-diagnostics`。
- runtime overview 新增 top-level `knowledge_pipelines`、summary 字段和 `Knowledge Pipelines` card。
- 抽出了共享 `AgentJob pressure -> worker coverage` helper，避免 runtime overview 与知识流水线聚合重复实现。

Tests run:
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 这层能力适合放在 Go，因为它只组合稳定 runtime state，不引入 LLM、RAG 策略或外部索引副作用。
- `knowledge-worker-diagnostics` 继续保留 job-type 视角；新增 `knowledge-pipeline-diagnostics` 提供 group/target 视角，两者互补，避免把单个 endpoint 做成混合语义。

Decision:
- 接受当前实现，保持只读，不让该聚合直接驱动 worker 调度、自动重试或 RAGFlow 写入。

Follow-ups:
- 如果后续需要更强 Go-owned knowledge orchestration，可在此基础上补 per-group checkpoint stagnation、dataset/index state 和 queue age。
