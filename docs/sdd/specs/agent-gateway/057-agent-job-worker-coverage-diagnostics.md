# 057 AgentJob Worker Coverage Diagnostics

Date: 2026-05-31

## 背景

上一轮已经把 Go-owned `AgentJob` pressure diagnostics 落到了 `/v1/job-metrics`
和 `/v1/runtime-overview`，可以看出哪类 job 在 pending / active 堆积。
但对运维和排障来说还差最后一层解释：

- 某类 job 堆积时，负责它的 Python worker 是否存在；
- worker 是否 active、只有 stale heartbeat、还是已经 failed；
- 当前高压 job type 是“单纯 backlog”还是“没有对应 worker 覆盖”。

这个诊断仍然属于 Go control-plane 的只读观测能力，因为它只组合：

- Go authoritative `AgentJob` pressure；
- Go authoritative Python worker heartbeat / lease status。

它不改变任何 Python 执行策略、并发、prompt 或 provider 逻辑。

## Go / Python 边界

Go 负责：

- 定义 `job_type -> expected worker_type[]` 的只读映射；
- 聚合 Go 已有 `AgentJob` pressure 与 `AgentWorkerStatuses`；
- 在 `/v1/runtime-overview` 暴露 worker coverage summary 和 card。

Python 负责：

- 继续上报 `agent-worker-statuses/report`；
- 继续执行 `group_memory_extract`、`rag_ingest`、`rag_eval`、
  `image_generation` 等 AI job；
- 不因为本切片自动调整 worker 并发、调度、优先级、retry 或 RAG 策略。

## 范围

- 新增 runtime-only 诊断视图 `AgentJobWorkerCoverageView`：
  - `job_type`
  - `expected_worker_types`
  - `high_pressure`
  - `pressure_reason`
  - `worker_count`
  - `active_workers`
  - `running_workers`
  - `failed_workers`
  - `stale_workers`
  - `coverage_status`
  - `coverage_reason`
- 只在 `runtime-overview` 聚合，不修改 `/v1/job-metrics` contract。
- 初始映射：
  - `group_memory_extract` -> `knowledge`
  - `rag_ingest` -> `knowledge`
  - `rag_eval` -> `rag_eval`
  - `image_generation` -> `image_generation`
- `active_workers = starting + idle + running`。
- `coverage_status` 规则：
  - `muted`: 当前 job type 没有已知 worker 映射；
  - `danger`: high pressure 且 `active_workers == 0`；
  - `warn`: 有 stale / failed worker，或未高压但 `active_workers == 0`；
  - `ok`: 至少一个 active worker，且无 stale / failed 告警。
- `/v1/runtime-overview` 新增 summary：
  - `agent_job_worker_coverage_job_types`
  - `agent_job_worker_coverage_uncovered_job_types`
  - `agent_job_worker_coverage_stale_job_types`
  - `agent_job_worker_coverage_failed_job_types`
- runtime overview 新增 `agent_job_worker_coverage` card。

## 不做

- 不改 `AgentJobMetricsService` 依赖边界，不把 worker 状态耦合进 `/v1/job-metrics`。
- 不改 Python worker 上报协议。
- 不做自动扩容、自动重启、自动切换 provider 或 worker 优先级调度。
- 不做 UI 大改，只补 runtime overview 聚合内容。

## 验收

- `RuntimeOverviewService` test 覆盖：
  - 高压 knowledge job 但无 active knowledge worker；
  - 有 stale / failed worker 时的 summary 和 card 状态；
  - 正常 coverage 时不误报。
- HTTP `runtime-overview` route test 覆盖新 summary/card 能正常返回。
- `go test ./app/service -run "TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./trigger/http -run "TestRuntimeOverviewEndpointReturnsGoOwnedAggregate" -count=1 -v` 通过。
- `go test ./...` 通过。
