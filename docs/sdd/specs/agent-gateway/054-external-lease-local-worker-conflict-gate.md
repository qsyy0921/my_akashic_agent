# 054 External Lease Local Worker Conflict Gate

Date: 2026-05-31

## 背景

Go runtime 已经有两个可能执行 outbox delivery side effect 的路径：

- local state-store outbox delivery worker；
- NATS JetStream `external_lease` outbox executor。

`startOutboxDeliveryWorker` 已经在启动阶段检查 external lease execution 是否已
ready，避免两个执行器同时运行。但 `/v1/queue-backend.external_lease` 的 cutover
gate 还没有提前暴露 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 这个冲突。
这会让配置诊断看起来 ready，直到真正启动 worker 才报错。

本切片把冲突检查前移到 queue backend gate：external lease cutover 只有在本地
outbox delivery worker 关闭时才能 ready。

## Go / Python 边界

Go 负责：

- 在 queue backend gate 中读取 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED`。
- 暴露 `local_outbox_worker_disabled` required check。
- 阻止 local worker 与 NATS external lease 同时拥有真实发送执行权。

Python 负责：

- 不参与 outbox delivery ownership 判断。
- 继续作为 AI worker 执行模型、Prompt、Memory/RAG、OCR/VLM 和图片生成。

## 范围

- `queueExternalLeaseGate` 增加 required check：
  - name: `local_outbox_worker_disabled`
  - passed: `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED` 不为 true
- 如果该检查失败：
  - `AllowExecution=false`
  - `GateState=blocked`
  - `ExecutionScope=none`
  - external lease consumer 不会启动
- 保留 `startOutboxDeliveryWorker` 里的 runtime 防御作为第二道保护。

## 不做

- 不改变 NATS consumer。
- 不改变 local outbox worker 实现。
- 不修改 Python worker。
- 不做真实平台发送 smoke。

## 验收

- 所有 base external lease cutover env 都通过，但 local outbox worker enabled 时，
  gate 仍然 blocked，且 blockers 包含 `local_outbox_worker_disabled`。
- local worker disabled 时，原有 external lease ready 测试不回退。
- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*ExternalLease" -count=1 -v` 通过。
- `go test ./...` 通过。
