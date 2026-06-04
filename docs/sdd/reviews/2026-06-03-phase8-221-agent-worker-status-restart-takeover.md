# Phase 8.221 Review - Agent Worker Status Restart Takeover

## What Changed

- Go `agent_worker_status` report 命令、DTO、handler 和 service 新增
  `replace_existing_instance_id`，允许在显式匹配现存实例 id 时做受控 takeover。
- Python `AgentWorkerStatusReporter` 在收到 `HTTP 409` 后，会解析
  `existing_instance_id`、提取旧 PID、确认旧 PID 已退出，并只重试一次 takeover。
- 新增 repo-owned live verifier
  `scripts/verify-agent-worker-status-restart.ps1`，并接入统一
  `scripts/verify-go-migration-goal.ps1`。

## Verification

- `uv run pytest tests/test_agent_gateway_worker_status.py tests/test_agent_gateway_client.py -q`
- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./app/service ./trigger/http`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-agent-worker-status-restart.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-go-migration-goal.ps1`

## Outcome

- Python 主进程重启后，新的 worker-status reporter 能在 stale-instance 场景下完成一次受控 takeover。
- fresh `logs/bot.log` 不再出现新的 `agent worker status lease conflict` 或
  `agent worker status report failed`。
- `/v1/agent-worker-statuses` 中 `image/knowledge/outbox` 的 active `instance_id`
  已切到当前 Python 主进程 PID。
- 这轮修复的是本地 runtime invariant，不改变 QQ 群静默、Telegram token 缺失、
  agent_job external lease blocked 等其它剩余 blocker。

