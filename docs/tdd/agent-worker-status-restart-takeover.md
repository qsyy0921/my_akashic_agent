# Agent Worker Status Restart Takeover TDD

## Go Tests

1. `services/agent-runtime/app/service/agent_worker_status_service_test.go`
   - 活跃 lease + 不同 `instance_id` 时仍返回冲突。
   - 当 `replace_existing_instance_id` 与当前记录完全匹配时，允许受控 takeover。
   - 当 `replace_existing_instance_id` 不匹配时，仍返回冲突。
2. `services/agent-runtime/trigger/http/handler_test.go`
   - `/v1/agent-worker-statuses/report` 默认冲突仍返回 `409`。
   - 带合法 `replace_existing_instance_id` 的重试请求可成功写入新实例状态。

## Python Tests

1. `tests/test_agent_gateway_worker_status.py`
   - 409 conflict + stale PID 时，reporter 只重试一次并带 `replace_existing_instance_id`。
   - 409 conflict + live PID 时，不做 takeover retry。
   - 无法解析 `existing_instance_id` 时，不做 takeover retry。
2. `tests/test_agent_gateway_client.py`
   - worker-status 上报 body 默认包含空的 `replace_existing_instance_id`。
   - takeover retry 时 body 带非空 `replace_existing_instance_id`。

## Live Verification

1. 重启 Python `main.py`。
2. 运行：
   - `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-worker-status-restart.ps1`
   - `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-go-migration-goal.ps1`
3. 期望：
   - fresh log 中没有新的 `agent worker status lease conflict`
   - fresh log 中没有新的 `agent worker status report failed`
   - `/v1/agent-worker-statuses` 里 `image/knowledge/outbox` 的 active `instance_id` 都切到当前 Python 主进程 PID
   - unified goal verifier 把 `agent_worker_status_restart` 标成 `live_verified`
