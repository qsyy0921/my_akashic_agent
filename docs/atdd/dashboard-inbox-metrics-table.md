# Dashboard Inbox Metrics Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `inbox_metrics` card 时，显示结构化 KPI、conversation 表格和 recent inbox event 表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `sampled_events/observe_only_total/reply_eligible_total/with_attachments/attachment_count/unique_senders`
   - `events_by_channel_kind`
   - `events_by_conversation` 的关键字段
   - `recent` 的关键字段
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.inbox_metrics_table=true` 纳入证据。
4. 整个过程保持只读，不发送 QQ/Telegram，不修改 observe/inbox 状态，不触发 Python AI。
