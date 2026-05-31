# 061 Knowledge Pipeline Job Lease Freshness Diagnostics

Date: 2026-05-31

## 背景

当前 `knowledge pipeline diagnostics` 已经覆盖：

- observe-only QQ 群 capture readiness
- `group_memory_extract` / `rag_ingest` backlog
- worker coverage
- checkpoint source lag
- checkpoint age / stagnant

但它仍缺少一个直接的执行态信号：当前 knowledge job 自己是不是已经卡在旧 lease 上。

目前如果一个群的 `group_memory_extract` 或 `rag_ingest` 任务已经：

- 处于 `leased/running` 很久；
- lease 已经过期但还没被恢复；
- active job 长时间没有 heartbeat / renew；

我们只能从 worker heartbeat、checkpoint lag 或单独查 `/v1/jobs` 间接推断。对 group-level control-plane 来说，这还不够直接。

这类诊断仍属于 Go-owned 确定性控制面，因为依赖的都是已有 AgentJob 状态：

- `status`
- `created_at`
- `updated_at`
- `lease_expires_at`

Python 不需要改变任何 memory / RAG 策略或执行协议。

## Go / Python 边界

Go 负责：

- 在 group-level knowledge pipeline 视图里暴露每个 stage 的 pending / active age；
- 判断 active lease 是否 stale 或 expired；
- 把 stage execution freshness 聚合进 pipeline status / reason；
- 在 runtime overview 暴露 lease freshness summary。

Python 负责：

- 继续通过 Go AgentJob lease / renew / running / succeeded / failed 协议执行 AI worker；
- 继续决定 memory 抽取、RAG ingest、chunking、embedding、retrieval 等策略；
- 不因为本切片改变 job 调度策略、重试策略或 worker autoscaling。

## 范围

- `KnowledgePipelineJobStageView` 新增：
  - `oldest_pending_age_seconds`
  - `oldest_active_age_seconds`
  - `stale_active_leases`
  - `expired_active_leases`
  - `freshness_status`
  - `freshness_reason`
- freshness 规则：
  - `expired_active_leases > 0` => `danger`, `expired_active_lease`
  - `stale_active_leases > 0` => `warn`, `stale_active_lease`
  - `pending > 0 && oldest_pending_age_seconds >= stale_after_seconds` => `warn`, `old_pending_backlog`
- pipeline 状态规则扩展：
  - `*_lease_expired` => pipeline `blocked`
  - `*_lease_stale` / `*_pending_old` => pipeline 至少 `warn`
- `KnowledgePipelineDiagnosticsView.Totals` 新增：
  - `expired_active_leases`
  - `stale_active_leases`
- `runtime-overview` 新增：
  - `knowledge_pipeline_expired_active_lease_targets`
  - `knowledge_pipeline_stale_active_lease_targets`

## 不做

- 不修改 AgentJob lease/recover/renew 协议。
- 不让 freshness 诊断自动恢复 lease、重试 job 或扩 worker。
- 不把 dataset/index metadata 拉进 Go。

## 验收

- service test 覆盖：
  - ready pipeline stage 暴露 age/freshness 默认值
  - stale active lease 触发 warn
  - expired active lease 触发 blocked
- runtime overview test 覆盖：
  - `knowledge_pipeline_expired_active_lease_targets`
  - `knowledge_pipeline_stale_active_lease_targets`
- HTTP endpoint test 至少验证 `freshness_status` / `expired_active_leases` 字段序列化存在。
- `go test ./app/service -run "TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestKnowledgePipelineDiagnosticsEndpointReturnsReadOnlyPipelines|TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
