# 180 Dashboard Outbox Metrics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `outbox_metrics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 outbox delivery throughput、dead-letter 最近样本和 channel-kind 分布。

## Decision

Python dashboard panel 为 `outbox_metrics` card 增加结构化只读 drilldown，直接展示：

- throughput 的 `queued/leased/dispatching/succeeded/failed/dead_lettered/terminal_events`
- dead letters current total
- `deliveries_by_channel_kind`
- recent dead-letter rows 的 `delivery_id/channel_kind/event_type/status/error_kind/error_message/attempt/occurred_at`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、租约或执行 outbox delivery
- 不修改 queue/backpressure 参数
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
