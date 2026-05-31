# 052 Queue Provider Capability Diagnostics

Date: 2026-05-31

## 背景

当前 Go runtime 已经支持本地 state-store 队列、NATS JetStream
`shadow_publish`、`dual_read_compare` 和受门禁保护的 `external_lease`。
用户侧已经明确要求“换成更合适的 MQ”并支持多线程消费。代码里实际已经把
NATS JetStream 作为第一阶段外部队列，但 `/v1/queue-backend` 只暴露
`supported_providers` 和 `recommended_first_backend`，没有解释各 provider
能力差异。

本切片把 MQ 选型边界做成 Go runtime 只读诊断能力：运行时明确告诉前端/运维
当前选择、推荐阶段、支持的迁移模式、并发消费能力、delayed nack 能力和后续
适配器状态。它不新增 Redis/RabbitMQ 适配器，不改变现有 NATS cutover gate，
也不触发真实平台发送。

## Go / Python 边界

Go 负责：

- 暴露 queue provider capability matrix。
- 描述当前 provider 的 shadow/dual-read/external-lease/result-ack 支持。
- 暴露 `goroutine_worker_pool` 并发消费与 `max_in_flight` 诊断。
- 把 Redis Streams / RabbitMQ 保留为后续 provider-neutral 边界。

Python 负责：

- 不参与 MQ provider 选择和队列租约控制。
- 继续作为 AI worker 执行模型、Prompt、Memory/RAG、OCR/VLM、图片生成和
  provider fallback。

## 范围

- `query.QueueBackendView` 增加：
  - `provider_capabilities`
  - `selected_provider_capability`
- `queueBackendViewFromEnv` 生成固定能力矩阵：
  - `local`: 当前本地 state-store，支持本地租约，不支持外部 MQ cutover。
  - `nats_jetstream`: 当前推荐第一外部 MQ，支持 shadow、dual-read、
    external lease、agent_job result-ack、多 goroutine consumer、delayed nack。
  - `redis_streams`: 后续候选，不在本轮实现外部 lease adapter。
  - `rabbitmq`: 后续候选，不在本轮实现外部 lease adapter。
- 测试覆盖默认 local 和 NATS provider 的能力摘要。

## 不做

- 不实现 Redis Streams / RabbitMQ adapter。
- 不改变 `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER` 等 cutover gate。
- 不修改 Python worker。
- 不做生产 MQ 性能压测。

## 验收

- `/v1/queue-backend` 可以展示当前 provider 能力摘要和 provider matrix。
- NATS JetStream 标记为当前推荐第一外部 MQ，且支持多线程消费和 delayed nack。
- Redis Streams / RabbitMQ 标记为 planned adapter，不能误报 external lease
  已可执行。
- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*Provider"` 通过。
- `go test ./...` 通过。
