# 046 Agent Worker Status Heartbeat Renewal

Date: 2026-05-31

## 背景

Phase 8.102 为 Go-owned Python AI worker status registry 增加了 `instance_id`
和 lease/fencing。该机制能防止同一 `worker_id` 的不同进程互相覆盖状态，但如果一个
Python worker 在长任务执行期间只上报一次 `running`，Go 侧 status lease 可能在任务仍
执行时过期，导致 runtime overview 误判 stale 或允许其它实例接管状态。

本切片补 Python worker status heartbeat：长任务运行期间周期性向 Go 上报 `running`，
刷新 worker status lease。它只续租 worker status，不改变 AgentJob lease token、
job heartbeat、ack/fail、MQ 消费语义。

## Go / Python 边界

Go 负责：

- 继续作为 worker status lease/fencing 的权威状态持有方。
- 通过既有 `/v1/agent-worker-statuses/report` 接受 `running` heartbeat。

Python 负责：

- `AgentWorkerStatusReporter` 提供 running heartbeat helper。
- image / knowledge / rag_eval / outbox worker 在处理长任务或发送任务期间启动 heartbeat。
- heartbeat 失败只记录 warning，不中断任务执行；真正 Job lease 仍由 `AgentJobLeaseHeartbeat` 或 outbox lease 负责。

## 范围

- 新增 `AgentWorkerStatusHeartbeat` helper，支持 `start()` / `stop()` / `beat_once()`。
- `AgentWorkerStatusReporter` 支持按 worker lease TTL 配置上报。
- image、knowledge、rag_eval worker 在 job running 期间续租 worker status。
- outbox worker 在 delivery dispatch 期间续租 worker status。

## 不做

- 不新增 Go endpoint。
- 不改变 AgentJob lease renewal。
- 不让 Go 管 Python 进程生命周期。

## 验收

- Python reporter tests 覆盖 heartbeat `beat_once()` 上报 `running/current_job_id/lease_ttl_seconds/instance_id`。
- 现有 image/knowledge/outbox/rag_eval worker tests 继续通过。
- `uv run pytest tests/test_agent_gateway_worker_status.py tests/test_agent_gateway_image_worker.py tests/test_agent_gateway_knowledge_worker.py tests/test_agent_gateway_outbox_worker.py tests/test_agent_gateway_rag_eval_worker.py -q` 通过。
- `go test ./...` 通过，证明 Go 侧未破坏。
