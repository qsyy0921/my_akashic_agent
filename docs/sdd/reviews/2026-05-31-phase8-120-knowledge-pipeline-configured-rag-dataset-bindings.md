# Review: phase8-120 knowledge pipeline configured rag dataset bindings

Spec:
- `docs/sdd/specs/agent-gateway/063-knowledge-pipeline-configured-rag-dataset-bindings.md`

Implementation summary:
- Python `QQGroupConfig` 新增可选 `ragflow_dataset_ids`，区分“未配置”与“显式空列表”。
- observe-only QQ 群在同步到 Go `observe_target.metadata` 时，现在会带上 `ragflow_dataset_ids` 和 `ragflow_dataset_count`。
- Python knowledge worker 现在支持按群覆盖 dataset 绑定，未显式配置的群继续回退到全局 `ragflow.default_dataset_ids`。
- Go `knowledge pipeline diagnostics` 会把配置期 dataset 与运行期 job/checkpoint dataset 合并；仅配置未启动的 dataset 显示为 `muted/configured_dataset_not_started`。
- runtime overview summary 新增 `knowledge_pipeline_configured_rag_datasets` 和 `knowledge_pipeline_configured_rag_dataset_not_started`。

Tests run:
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests/test_bootstrap_wiring_p2.py -k "group_ragflow_dataset_ids or builds_agent_runtime_observe_targets"`
- `New-Item -ItemType Directory -Force .tmp\pytest | Out-Null; $env:TMP=(Resolve-Path .tmp\pytest).Path; $env:TEMP=$env:TMP; uv run pytest tests/test_agent_gateway_knowledge_worker.py -k "group_specific_dataset_bindings or enqueues_group_memory_and_rag_jobs"`
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v`
- `go test ./...`

Findings:
- 直接 `python -m pytest` 失败不是代码问题，而是本机环境缺少兼容 `pytest.ini` 的 `pytest-asyncio`；项目级 `uv run pytest` 正常。
- 直接使用系统临时目录时，Windows 当前用户对默认 `pytest-of-*` 临时目录存在 `PermissionError`；本轮通过将 `TMP/TEMP` 定向到仓库内 `.tmp\pytest` 规避。

Decision:
- 接受。该切片把“应存在的数据集绑定”提升为 Go 可见的稳定控制面元数据，没有侵入 Python 的 RAG 策略和外部索引执行。

Follow-ups:
- 后续若需要更强索引面观察，可继续把 dataset/index readiness、文档数、最近 ingest 时间等只读元数据接入 Go。
