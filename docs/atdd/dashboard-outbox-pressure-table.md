# Dashboard Outbox Pressure Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `outbox_pressure` card 时，显示结构化 KPI 和账号表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `accounts/high_pressure/max_queued/max_active`
   - throughput 的 `queued/leased/succeeded`
   - dead letters current total
   - 每个账号的 `account_key/channel/account_id/queued/dispatching/active/dead_lettered/high_pressure/reason`
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.outbox_pressure_table=true` 纳入证据。
4. 整个过程保持只读，不发送 QQ/Telegram，不租约 outbox，不触发 Python AI。
