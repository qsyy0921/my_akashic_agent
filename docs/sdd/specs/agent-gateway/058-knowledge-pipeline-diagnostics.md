# 058 Knowledge Pipeline Diagnostics

Date: 2026-05-31

## 背景

当前 Go runtime 已分别持有：

- observe-only QQ 群目标；
- inbox / media capture coverage；
- generic `AgentJob` 生命周期与 pressure；
- Python worker heartbeat；
- knowledge checkpoints。

但这些状态仍分散在多个 endpoint 中，无法直接回答：

- 某个观察群当前的群记忆流水线是否 ready；
- `group_memory_extract` 与 `rag_ingest` 在这个群里是否积压；
- 该群当前是 capture 缺失、checkpoint 未推进，还是 worker 缺席导致的阻塞。

这类“群级知识流水线状态”属于稳定 control-plane 编排与诊断，适合沉淀到 Go；
而 chunking、embedding、retrieval、RAG answer synthesis 仍留在 Python。

## Go / Python 边界

Go 负责：

- 聚合 observe target、capture、group-level job 状态、checkpoint、worker coverage。
- 暴露只读 `knowledge pipeline diagnostics` endpoint。
- 在 `runtime-overview` 聚合群级 pipeline readiness summary/card。

Python 负责：

- 继续执行 `group_memory_extract`、`rag_ingest` 等 AI worker 任务。
- 继续决定 RAGFlow 上传、chunking、embedding、检索、总结和沉淀策略。
- 不因为本切片自动修改 worker 并发、job 提交或 AI 策略。

## 范围

- 新增 `KnowledgePipelineDiagnosticsView`，按 observe-only QQ group 返回：
  - `target_id`
  - `channel`
  - `enabled`
  - `observe_only`
  - `capture_status`
  - `receiver_connected`
  - `capture_blockers`
  - `group_memory`
  - `rag_ingest`
  - `memory_checkpoint`
  - `rag_checkpoints`
  - `worker_coverage`
  - `status`
  - `reasons`
- `group_memory` / `rag_ingest` 聚合至少包含：
  - `pending`
  - `leased`
  - `running`
  - `active`
  - `latest_job`
- `worker_coverage` 复用 Go 已有 `AgentJob pressure -> worker coverage` 只读逻辑，不引入新的 Python 协议。
- 新增 `GET /v1/knowledge-pipeline-diagnostics`。
- `runtime-overview` 新增：
  - top-level `knowledge_pipelines`
  - summary 字段 `knowledge_pipeline_targets` / `ready` / `warning` / `blocked`
  - `Knowledge Pipelines` card。

## 状态规则

- `status=blocked`：
  - target enabled 但 capture 已 blocked；或
  - `group_memory_extract` / `rag_ingest` 出现 high pressure 且无 active worker。
- `status=warn`：
  - capture warn；或
  - 有 knowledge job pending / active 但 checkpoint 长时间未出现；或
  - worker coverage 为 warn。
- `status=ok`：
  - target enabled；
  - receiver/capture ready；
  - 没有 blocked / warn 条件。
- `status=muted`：
  - target 未启用。

## 不做

- 不把 RAGFlow 数据集管理、上传、解析迁到 Go。
- 不改变 Python knowledge worker 提交/执行逻辑。
- 不新增自动重试策略、优先级调度、worker autoscaling。
- 不做前端大改，只提供 runtime overview 聚合数据。

## 验收

- 新 service test 覆盖：
  - 正常 ready 群；
  - capture blocked 群；
  - high-pressure + no active worker 群。
- HTTP route test 覆盖 `/v1/knowledge-pipeline-diagnostics` 返回只读聚合。
- runtime overview test 覆盖新的 summary/card。
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
