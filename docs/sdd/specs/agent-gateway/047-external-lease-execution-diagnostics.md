# 047 External Lease Execution Diagnostics

Date: 2026-05-31

## 背景

`external_lease` 已经能让 Go runtime 消费 NATS work notification，并把
`outbox_delivery` / `agent_job` 的结果映射到 queue `ack` / `nack` / `term`。
但目前 `/v1/queue-backend` 只能看到 gate、shadow publish、dual read compare
诊断，不能直接看到 external lease 实际执行后的 disposition 分布、失败原因和最近样本。

这会影响 MQ cutover 前的可观测性：我们需要证明 Go 对外部队列的处理仍以 Go state
store 为权威，并且 NATS ack/nack/term 与 outbox / AgentJob 状态一致。

## Go / Python 边界

Go 负责：

- 记录 external lease 每次执行的 bounded diagnostic sample。
- 统计 disposition、reason、work_kind。
- 在 `/v1/queue-backend` 的 `external_lease.diagnostics` 暴露只读快照。

Python 负责：

- 继续执行 `agent_job` 的 AI 工作：模型调用、Memory/RAG、图片生成、OCR/VLM。
- 通过既有 AgentJob API 回写 `running` / `succeeded` / `failed`。

## 范围

- 给 `WorkQueueExternalLeaseService` 增加内存诊断 recorder。
- 每次 `ExecuteWorkQueueLease` 返回时记录：
  - `work_kind`
  - `work_id`
  - `subject`
  - `disposition`
  - `reason`
  - `state_status`
  - `attempts`
  - `executed_at`
- `/v1/queue-backend` 在 `external_lease` gate 下返回 diagnostics。
- diagnostics 只读，不触发 lease、send、ack 或 Python worker。

## 不做

- 不新增平台发送 cutover。
- 不新增真实 MQ provider。
- 不让 Go 执行 Python AI job。
- 不持久化 diagnostics；它是进程内运行态观察，持久事实仍来自 outbox/job state 与 event stream。

## 验收

- Go service tests 覆盖 external lease diagnostics 记录 ack/nack/term 和最近样本。
- Queue backend service tests 覆盖 `external_lease.diagnostics` 快照接入。
- `go test ./...` 通过。
