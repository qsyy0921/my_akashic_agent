# Dashboard Observe Capture Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `observe_capture` card 时，显示结构化 KPI、target 表格和 notes，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - `targets/ready/warning/blocked/image_covered/file_covered/content_ready_assets/receiver_activity_recent`
   - `targets` 的关键字段
   - `notes`
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.observe_capture_table=true` 纳入证据。
4. 整个过程保持只读，不修改 observe capture / media 状态，不发送 QQ/Telegram，不触发 Python AI。
