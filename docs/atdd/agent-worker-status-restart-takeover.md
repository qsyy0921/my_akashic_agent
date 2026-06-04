# Agent Worker Status Restart Takeover ATDD

## Scenario

本地 Python `main.py` 重启后，Go runtime 中旧的 worker-status lease 仍未过期，
新的 Python worker reporter 需要在确认旧 PID 已退出后完成一次受控 takeover，
而不是持续报 409 conflict。

## Preconditions

- Go runtime 运行在 `http://127.0.0.1:8780`
- Python `main.py` 使用 repo 推荐入口重启
- QQ 群发保持关闭

## Acceptance Checks

1. 运行 `uv run pytest tests/test_agent_gateway_worker_status.py tests/test_agent_gateway_client.py -q`
   与 `go test ./app/service ./trigger/http`，确认：
   - Go 仍默认拒绝活跃 lease 冲突；
   - Python reporter 仅在 stale-instance 场景下做一次 takeover retry。
2. 重启 Python `main.py` 后运行：
   - `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-worker-status-restart.ps1`
3. 验证脚本必须返回：
   - `conclusion.status = live_verified`
   - `checks.no_worker_status_conflict_lines = true`
   - `checks.no_worker_status_report_failed_lines = true`
   - `checks.active_worker_instance_ids_match_current_port_owner = true`
4. `/v1/agent-worker-statuses` 中 `image/knowledge/outbox` 三个 active worker 的
   `instance_id` 必须切到当前 Python 主进程 PID。

## Failure Signals

- fresh log 中出现新的 `agent worker status lease conflict`
- fresh log 中出现新的 `agent worker status report failed`
- active worker `instance_id` 仍指向旧 PID
- Go handler 不再对普通冲突返回 409
