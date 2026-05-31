# Phase 8.102 Review: Agent Worker Status Lease Fencing

日期：2026-05-31

## 结论

通过。该切片为 Go-owned Python AI worker status registry 增加 `instance_id`、
`lease_until` 和 lease conflict 检查，防止同一 `worker_id` 的多个 Python 进程互相覆盖
dashboard/runtime overview 状态。它只保护状态所有权，不改变 AgentJob lease token、
heartbeat、ack/fail 或 MQ 消费语义。

## 设计审查

- `AgentWorkerStatus` 领域模型增加 lease 字段和 `LeaseActive()` 判断，fencing 规则在 app service 内完成。
- HTTP report 接口新增 `instance_id` / `lease_ttl_seconds`，活跃 lease 冲突返回 409。
- Python `AgentWorkerStatusReporter` 每个进程生成稳定 instance id，并随 status report 上报。
- 旧客户端不带 `instance_id` 时仍可兼容；新客户端会获得 Go-side fencing。
- 没有新增服务，继续复用 `services/agent-runtime` 的 DDD/六边形分层。

## 风险

- 本切片不会主动停止第二个 Python 进程；它只阻止 status 覆盖并记录 warning。后续如果需要进程级控制，需要独立 worker control 协议。
- 如果两个进程使用不同 `worker_id`，本 fencing 不会拦截；这是预期，因为 worker_id 是显式身份边界。
- `lease_ttl_seconds` 过短会导致状态接管过快，过长会导致异常进程占位更久；当前默认 120 秒。

## 验收门

- `go test ./domain/model ./app/service ./infrastructure/agentworkerstatusstore ./trigger/http`
- `uv run pytest tests/test_agent_gateway_client.py tests/test_agent_gateway_image_worker.py -q --basetemp .tmp/pytest-worker-status-lease`
- `uv run python -m py_compile integrations/agent_gateway.py integrations/agent_gateway_worker_status.py`
- 后续 live 验证：同一 `worker_id` 双进程上报时第二个进程收到 409，原状态未被覆盖。
