# 056 AgentJob Pressure Diagnostics

Date: 2026-05-31

## 背景

Go runtime 现在已经持有 generic `AgentJob` 的权威生命周期、租约、重试、dead-letter、
event stream 和 knowledge worker diagnostics。
但当前 `AgentJobMetrics` 仍主要回答“发生了多少事件、现在是什么状态”，不能直接回答：

- `group_memory_extract` / `rag_ingest` 是否在堆积；
- 哪类 job 有大量 pending / active backlog；
- 某类 job 是否长时间 pending，可能意味着 Python worker 不足或执行停滞。

本切片先做 Go-owned、只读的 job-type pressure diagnostics，作为 knowledge / RAG /
future AI worker 调度的基础设施观察面，不改变 Python worker 的执行职责。

## Go / Python 边界

Go 负责：

- 从 Go-owned `AgentJob` state 计算 job-type backlog / pressure。
- 在 `/v1/job-metrics` 暴露 read-only pressure 统计。
- 在 `/v1/runtime-overview` summary/card 中聚合 pressure 结果。

Python 负责：

- 继续作为 AI worker 执行 `group_memory_extract`、`rag_ingest`、`rag_eval`、
  image generation、OCR/VLM 等 AI job。
- 不因为本切片自动改变 lease、执行优先级、并发度或 prompt 策略。

## 范围

- `AgentJobMetricsFilter` 增加可选 `Now`，便于稳定测试 oldest age 计算。
- `AgentJobMetricsView` 增加 `pressure` 字段：
  - `job_types`
  - `high_pressure_job_types`
  - `max_pending`
  - `max_active`
  - `by_type`
- `AgentJobTypePressureView` 聚合：
  - `job_type`
  - `pending`
  - `leased`
  - `running`
  - `active`
  - `oldest_pending_age_seconds`
  - `high_pressure`
  - `pressure_reason`
- pressure 聚合以 `job_type` 为 key。
- `active = leased + running`。
- 初始只读 high pressure 阈值：
  - `pending >= 10`
  - `active >= 5`
  - `oldest_pending_age_seconds >= 900`
- `/v1/runtime-overview` 增加：
  - `agent_job_pressure_job_types`
  - `agent_job_pressure_high_job_types`
  - `agent_job_pressure_max_pending`
  - `agent_job_pressure_max_active`
  - `agent_job_pressure_oldest_pending_age_seconds`
- runtime overview 增加 `agent_job_pressure` card。

## 不做

- 不修改 Python knowledge worker / image worker / rag_eval worker。
- 不改变 Go `AgentJob` lease、retry、recover-expired 或 external lease 语义。
- 不新增 worker autoscaling、优先级调度、拒绝 enqueue、限流或 cutover 逻辑。
- 不把压力诊断直接绑定到 prompt 或 RAG 策略。

## 验收

- Go `AgentJobMetricsService` test 覆盖 job-type pressure 聚合、oldest pending age 和 high pressure 判断。
- Go `RuntimeOverviewService` test 覆盖 summary 和 `agent_job_pressure` card。
- `go test ./app/service -run "TestAgentJobMetricsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v` 通过。
- `go test ./...` 通过。
