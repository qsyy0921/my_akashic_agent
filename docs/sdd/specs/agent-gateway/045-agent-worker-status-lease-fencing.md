# 045 Agent Worker Status Lease Fencing

Date: 2026-05-31

## 背景

Go 已经持有 Python AI worker status registry，用于 dashboard/runtime overview 观察
image、knowledge、rag_eval、outbox 等 worker 的 `starting|idle|running|failed|stopped`
状态。当前状态按 `worker_id` 覆盖写入：如果两个 Python 进程使用同一个 `worker_id`，
后启动或误启动的进程会覆盖前一个进程的状态，dashboard 无法判断真实 owner。

本切片为 status registry 增加轻量 lease/fencing。它不是 AgentJob lease 的替代，
只保护 worker status 记录的 owner，不改变 job lease token、heartbeat、ack/fail 协议。

## Go / Python 边界

Go 负责：

- 保存 worker status 的 `instance_id`、`lease_until`、`lease_active`。
- 在同一个 `worker_id` 已有活跃 lease 时，拒绝不同 `instance_id` 的状态覆盖。
- stale read 时继续按 heartbeat 规则标记 stopped/stale。

Python 负责：

- 每个 worker reporter 进程生成稳定 `instance_id`。
- 每次 status report 携带 `instance_id` 和 `lease_ttl_seconds`。
- 收到 fencing conflict 时记录告警；worker 具体是否退出由后续 worker control 切片决定。

## 范围

Go：

- `AgentWorkerStatus` domain model 增加 `InstanceID`、`LeaseUntil`。
- `ReportAgentWorkerStatusCommand` / DTO 增加 `instance_id`、`lease_ttl_seconds`。
- `AgentWorkerStatusView` 增加 `instance_id`、`lease_until`、`lease_active`。
- `AgentWorkerStatusService` 在活跃 lease 内拒绝不同实例覆盖。

Python：

- `AgentWorkerStatusReporter` 生成 instance id，并传给 `AgentGatewayClient.report_agent_worker_status()`。
- `AgentGatewayClient.report_agent_worker_status()` 发送新增字段。

## 不做

- 不改变 `/v1/jobs/lease-next`、job lease token、result ack/fail 语义。
- 不让 Go 停止 Python 进程；本切片只做状态 fencing 和诊断。
- 不新增服务，继续复用 `services/agent-runtime` 现有 DDD 分层。

## 验收

- Go service/http/store tests 覆盖同 worker_id 异 instance 活跃 lease conflict。
- Python client/reporter tests 覆盖 instance_id 和 lease_ttl_seconds 上报。
- `go test ./app/service ./infrastructure/agentworkerstatusstore ./trigger/http` 通过。
- `uv run pytest tests/test_agent_gateway_client.py` 通过。
