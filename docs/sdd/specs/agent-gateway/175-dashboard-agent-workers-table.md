# 175 Dashboard Agent Workers Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `agent_workers` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 Python worker 的 stale/idle/backoff 状态、lease 活跃度和当前 worker type 分布。

## Decision

Python dashboard panel 为 `agent_workers` card 增加结构化只读 drilldown，直接展示：

- `workers/idle/running/stopped/stale/failed/knowledge/outbox_delivery`
- 每个 worker 的 `worker_id/worker_type/status/lease_active/stale/processed_total/failed_total/reason`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不启动、停止、重启或替换 Python worker
- 不修改 worker lease
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
