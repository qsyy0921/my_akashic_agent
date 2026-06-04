# Dashboard Delivery Adapters Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `delivery_adapters` card 时，显示结构化 adapter 表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `provider/channel/transport/enabled/endpoint_configured/access_token_configured/endpoint`
3. 现有 `Adapter Health` 和 `Delivery Smoke` 入口仍保留。
4. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.delivery_adapters_table=true` 纳入证据。
5. 整个过程保持只读，不修改 adapter 配置，不发送 QQ/Telegram，不触发 Python AI。
