# Dashboard Outbox Metrics Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `outbox_metrics` card 时，显示结构化 KPI、channel 分布和 recent dead-letter 表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - throughput 的 `queued/leased/dispatching/succeeded/failed/dead_lettered/terminal_events`
   - dead letters current total
   - `deliveries_by_channel_kind`
   - recent dead-letter rows 的关键字段
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.outbox_metrics_table=true` 纳入证据。
4. 整个过程保持只读，不发送 QQ/Telegram，不租约 outbox，不触发 Python AI。
