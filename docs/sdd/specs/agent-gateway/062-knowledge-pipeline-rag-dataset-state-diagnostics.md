# 062 Knowledge Pipeline RAG Dataset State Diagnostics

Date: 2026-05-31

## 背景

当前 `knowledge pipeline diagnostics` 已经能从群级视角回答：

- capture 是否 ready
- `group_memory_extract` / `rag_ingest` 是否堆积
- worker coverage 是否缺席
- checkpoint source lag / age / stagnation
- stage lease freshness 是否 stale / expired

但对 RAG 来说，群级视图仍然太粗。一个群往往会绑定多个 dataset，而当前 Go control-plane 只能看到：

- `rag_ingest` 这个 stage 是否整体有问题
- `rag_checkpoint_lag_max` 最大落后值

它还不能直接回答：

- 到底是哪个 dataset 在落后、卡住或 lease 过期；
- 某个 dataset 当前是否有 pending / active `rag_ingest` job；
- 某个 dataset 的 checkpoint / lag / freshness 是不是已经异常。

这些信息已经稳定存在于 Go-owned 状态中：

- `rag_ingest` job payload / route / lifecycle
- `ragflow:*` checkpoint metadata

因此可以继续往前推进为 Go 侧只读 dataset control-plane，而不用触碰 Python RAGFlow 执行逻辑。

## Go / Python 边界

Go 负责：

- 基于 `rag_ingest` job payload 中的 `dataset_id` 分组；
- 基于 `ragflow:*` checkpoint metadata 中的 `dataset_id` 分组；
- 为每个 observe-only QQ 群输出 per-dataset state；
- 在 runtime overview 聚合 dataset-level degraded counts。

Python 负责：

- 继续调用 RAGFlow API 执行 upload / parse / retrieve；
- 继续决定 chunking、embedding、retrieval、rerank、answer synthesis；
- 不因为本切片改变 dataset 选择策略、索引策略或 worker 执行协议。

## 范围

- `KnowledgePipelineView` 新增：
  - `rag_datasets`
- 新增 `KnowledgePipelineRagDatasetView`：
  - `dataset_id`
  - `display_name`
  - `job_stage`
  - `checkpoint`
  - `checkpoint_lag`
  - `status`
  - `reasons`
- dataset 状态来源：
  - `job_stage` 复用现有 `KnowledgePipelineJobStageView`
  - `checkpoint_lag` 复用现有 `KnowledgePipelineCheckpointLagView`
- dataset 状态规则：
  - stage freshness `expired_active_lease` => `blocked`
  - stage freshness `stale_active_lease` / `old_pending_backlog` => `warn`
  - checkpoint lag/stagnant/stalled 复用现有 rag 规则
- `KnowledgePipelineDiagnosticsView.Totals` 新增：
  - `rag_datasets`
  - `rag_dataset_warning`
  - `rag_dataset_blocked`
- `runtime-overview` 新增：
  - `knowledge_pipeline_rag_datasets`
  - `knowledge_pipeline_rag_dataset_warning`
  - `knowledge_pipeline_rag_dataset_blocked`

## 不做

- 不调用外部 RAGFlow API 查询 dataset parse 状态。
- 不改变 `rag_ingest` job payload / checkpoint 协议。
- 不把 dataset state 用作自动调度、副作用或 worker 扩缩容依据。

## 验收

- service test 覆盖：
  - ready pipeline 能输出 `rag_datasets`
  - 单 dataset lag/stagnant 能落在对应 dataset 上
  - 单 dataset expired lease 能落在对应 dataset 上并 blocked
- runtime overview test 覆盖：
  - `knowledge_pipeline_rag_datasets`
  - `knowledge_pipeline_rag_dataset_warning`
  - `knowledge_pipeline_rag_dataset_blocked`
- HTTP endpoint test 至少验证 `rag_datasets[].dataset_id` 字段序列化存在。
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
