# 182 Dashboard Inbox Metrics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `inbox_metrics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 observe-only 捕获量、attachment 覆盖、conversation 分布和最近接收事件。

## Decision

Python dashboard panel 为 `inbox_metrics` card 增加结构化只读 drilldown，直接展示：

- `sampled_events/observe_only_total/reply_eligible_total/with_attachments/attachment_count/unique_senders`
- `events_by_channel_kind`
- `events_by_conversation` 的 `conversation/channel/total/observe_only/reply_eligible/with_attachments/unique_senders/latest_seq/latest_received_at`
- `recent` 的 `event_id/channel/sender_kind/decision_action/observe_only/attachment_count/seq/received_at`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、租约或执行 outbox delivery
- 不修改 observe target / receiver / checkpoint 状态
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
