# 179 Dashboard Outbox Pressure Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `outbox_pressure` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查账号级 queued/active 压力、throughput 和 dead-letter 基线。

## Decision

Python dashboard panel 为 `outbox_pressure` card 增加结构化只读 drilldown，直接展示：

- `accounts/high_pressure_accounts/max_queued/max_active`
- throughput 的 `queued/leased/succeeded`
- dead letters current total
- `pressure.by_account` 的 `account_key/channel_kind/account_id/queued/dispatching/active/dead_lettered/high_pressure/pressure_reason`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、租约或执行 outbox delivery
- 不修改 queue/backpressure 参数
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
