# 060 Knowledge Pipeline Checkpoint Age Diagnostics

Date: 2026-05-31

## 背景

上一轮已经把 `knowledge pipeline diagnostics` 推进到 source-seq lag 视角：

- Go 能看见 observe-only QQ 群最新 `metadata.seq`
- Go 能看见 memory / rag checkpoint cursor
- Go 能判断 lagging 和 stalled-under-pressure

但当前视图还缺少时间维度。它还不能稳定回答：

- 某个 checkpoint 虽然有 cursor，但它多久没有推进了；
- 当前是“短暂落后”，还是“已经停在老 checkpoint 上很久”；
- 群知识流水线 blocked/warn 时，时间维度是否已经说明它进入了真正的 stagnation。

这仍然属于 Go-owned 只读控制面，因为依赖的都是确定性状态：

- checkpoint `updated_at`
- 当前诊断请求时间 `now`
- source lag 和 Go 侧 job pressure / worker coverage

Python 不需要为此改变任何 RAG / memory 执行策略。

## Go / Python 边界

Go 负责：

- 基于 checkpoint `updated_at` 计算 checkpoint `age_seconds`；
- 在 knowledge pipeline 视图里暴露 memory / rag 的 checkpoint age；
- 将“lag + age”组合成更可靠的 stagnation 诊断；
- 在 runtime overview 聚合 `stale_checkpoint_targets` / `stagnant_checkpoint_targets`。

Python 负责：

- 继续执行 `group_memory_extract`、`rag_ingest`；
- 继续决定 chunking、embedding、retrieval、rerank、answer synthesis；
- 不因为本切片自动改变重试、并发、worker autoscaling 或 checkpoint 推进策略。

## 范围

- `KnowledgePipelineCheckpointLagView` 新增：
  - `age_seconds`
- `KnowledgePipelineView` 继续复用现有 `memory_checkpoint_lag` / `rag_checkpoint_lag_max`，不再新造平行结构。
- 新增 checkpoint age/stagnation 规则：
  - `age_seconds >= stale_after_seconds` 视为 checkpoint stale；
  - `lag warn/danger` 且 `age_seconds >= stale_after_seconds`，记入 `*_checkpoint_stagnant` reason，并将 pipeline 至少提升为 `warn`；
  - `lag danger` 且 `age_seconds >= stale_after_seconds` 且对应 stage high pressure，继续保留更强的 `*_checkpoint_stalled_under_pressure` blocked 语义。
- `KnowledgePipelineDiagnosticsView.Totals` 新增：
  - `stale_checkpoints`
  - `stagnant`
- `runtime-overview` 新增：
  - `knowledge_pipeline_stale_checkpoint_targets`
  - `knowledge_pipeline_stagnant_targets`

## 不做

- 不把 dataset/index metadata 拉进 Go。
- 不把 checkpoint age 诊断直接转成调度副作用。
- 不改变 Python worker heartbeat / checkpoint upsert 协议。

## 验收

- service test 覆盖：
  - ready pipeline 暴露 checkpoint `age_seconds`
  - lag + age 触发 `*_checkpoint_stagnant`
  - stalled-under-pressure 仍优先保持 blocked 语义
- runtime overview test 覆盖：
  - `knowledge_pipeline_stale_checkpoint_targets`
  - `knowledge_pipeline_stagnant_targets`
- HTTP endpoint test 至少验证 `age_seconds` 字段序列化存在。
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
