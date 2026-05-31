# 059 Knowledge Pipeline Source Lag Diagnostics

Date: 2026-05-31

## 背景

上一轮已经把 `knowledge pipeline diagnostics` 做成了按 observe-only QQ 群聚合的
control-plane 视图，可以看到 capture、knowledge jobs、checkpoint 和 worker
coverage。

但它还不能回答一个关键问题：

- 当前 checkpoint 相对群消息源到底落后了多少；
- 是“有 checkpoint 但没跟上 source seq”，还是“checkpoint 已经对齐”；
- 在高压知识流水线下，是否已经出现 checkpoint 实质停滞。

这类 source lag 仍属于 Go-owned 确定性诊断，因为比较的两端都已经在 Go 中：

- inbox observe-only 原始消息及其 `metadata.seq`
- knowledge checkpoint cursor

Python 继续负责 chunking / embedding / retrieval / answer synthesis。

## Go / Python 边界

Go 负责：

- 读取 observe-only inbox 事件中的 `metadata.seq`；
- 计算每个 observe-only QQ 群的 `latest_source_seq`；
- 对比 memory / rag checkpoint cursor，形成 lag / stalled 诊断；
- 在 runtime overview 暴露 lagging / stalled summary。

Python 负责：

- 继续提交/执行 `group_memory_extract`、`rag_ingest`；
- 继续决定 chunking、embedding、retrieval、RAG answer 策略；
- 不因为本切片自动更改 job 调度、worker 并发或 checkpoint 推进策略。

## 范围

- `KnowledgePipelineView` 新增：
  - `sequenced_events`
  - `source_seq_known`
  - `latest_source_seq`
  - `memory_checkpoint_lag`
  - `rag_checkpoint_lag_max`
- 新增 `KnowledgePipelineCheckpointLagView`：
  - `checkpoint_id`
  - `cursor`
  - `latest_source_seq`
  - `lag`
  - `status`
  - `reason`
  - `updated_at`
- `KnowledgePipelineDiagnosticsService` 增加 inbox 依赖，按 target 读取 bounded inbox sample 并计算最大 `metadata.seq`。
- lag 阈值：
  - `warn`: `lag >= 20`
  - `danger`: `lag >= 100`
- pipeline 状态规则扩展：
  - lag `warn` 记入 `*_checkpoint_lagging` reason，并把 pipeline 至少提升为 `warn`；
  - lag `danger` 且对应 stage 已 high pressure 时，记入 `*_checkpoint_stalled_under_pressure`，并把 pipeline 提升为 `blocked`。
- `runtime-overview` 新增：
  - `knowledge_pipeline_lagging_targets`
  - `knowledge_pipeline_stalled_targets`

## 不做

- 不把具体 dataset/index state 拉进 Go。
- 不让 lag 诊断直接触发自动重试、补提 job、worker autoscaling 或 checkpoint 修复。
- 不修改 Python inbox / checkpoint 写入协议，只复用现有 `seq` 和 cursor。

## 验收

- service test 覆盖：
  - source seq 与 checkpoint 对齐；
  - lag warn；
  - high pressure + lag danger 触发 stalled blocked。
- runtime overview test 覆盖 lagging / stalled summary。
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
