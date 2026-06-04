# 176 Dashboard Runtime Workers Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `runtime_workers` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查当前 Go worker 的启停状态、poll 间隔、batch/concurrency 配置和 worker boundary。

## Decision

Python dashboard panel 为 `runtime_workers` card 增加结构化只读 drilldown，直接展示：

- `workers/enabled/disabled/running`
- 每个 runtime worker 的 `name/kind/enabled/running/worker_id/interval_or_concurrency/batch_or_max_in_flight/notes`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不启动、停止或重启 Go worker
- 不修改 worker interval、batch、consumer concurrency 或 queue flags
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
