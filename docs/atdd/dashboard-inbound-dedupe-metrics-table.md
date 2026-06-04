# Dashboard Inbound Dedupe Metrics Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `inbound_dedupe_metrics` card 时，显示结构化 KPI、scope 表格和 notes，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `sampled_records/active_records/expired_records/duplicate_records/seen_total/duplicate_seen_total`
   - `scopes` 的关键字段
   - `notes`
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.inbound_dedupe_metrics_table=true` 纳入证据。
4. 整个过程保持只读，不修改 dedupe 记录，不发送 QQ/Telegram，不触发 Python AI。
