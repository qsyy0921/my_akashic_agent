# Review: phase8-121 knowledge pipeline rag ingest snapshot diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/064-knowledge-pipeline-rag-ingest-snapshot-diagnostics.md`

Implementation summary:
- Python `AgentGatewayKnowledgeWorker` 在成功 `rag_ingest` 后，把稳定摘要写入 Go knowledge checkpoint metadata：`last_message_count`、`last_document_count`、`last_start_seq`、`last_end_seq`、`last_parse_requested`，并继续保留 `display_name`。
- Go `knowledge pipeline diagnostics` 新增结构化 `ingest_snapshot`，按 dataset 暴露最近一次 ingest 的摘要，不再要求前端解析原始 checkpoint metadata。
- runtime overview summary 新增 `knowledge_pipeline_rag_dataset_ingest_snapshots`，作为只读聚合计数。

Tests run:
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests/test_agent_gateway_knowledge_worker.py -k "processes_rag_ingest_job or group_specific_dataset_bindings"`
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests/test_agent_gateway_knowledge_worker_smoke.py -k "smoke_knowledge_worker_end_to_end_with_real_group_memory_and_fake_ragflow"`
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 适合沉到 Go 的是“最近一次成功 ingest 的稳定摘要”，不适合沉的是完整 document id 列表和 provider-specific parse payload；后者仍应留在 Python job result。
- 通过 checkpoint metadata 持久化摘要，比依赖 bounded job result sample 更稳定，因为 checkpoint 已是 Go 的长期控制面状态。

Decision:
- 接受。该切片继续强化了 Go control-plane 的可观测性，没有扩大到 Python RAG 算法实现或外部服务写路径。

Follow-ups:
- 后续若需要更细观察，可继续把最近 parse readiness 或累计文档量接入 Go，但要保持 provider-neutral 和只读。
