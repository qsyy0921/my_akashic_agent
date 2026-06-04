# Dashboard Send Ledger Metrics Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `send_ledger_metrics` card 时，显示结构化 KPI、repeated hashes 表格和 recent records 表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `sampled_records/unique_bots/unique_conversations/unique_content_hashes/repeated_content_hashes`
   - `repeated_hashes` 的关键字段
   - `recent` 的关键字段
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.send_ledger_metrics_table=true` 纳入证据。
4. 整个过程保持只读，不发送 QQ/Telegram，不修改 send ledger，不触发 Python AI。
