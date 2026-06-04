# Dashboard Agent Job Metrics Table ATDD

## Acceptance

1. `runtime_overview` panel 打开 `agent_job_metrics` card 时，显示结构化表格，而不是只剩 raw JSON。
2. drilldown 至少能直接读到：
   - sampled jobs / events
   - throughput `created/leased/running/succeeded`
   - `jobs_by_type`
   - `pressure.by_type`
   - `dead_letters.recent`
3. `scripts/verify-go-migration-goal.ps1` 当前 turn 会把 `dashboard_read_models.agent_job_metrics_table=true` 纳入证据。
4. 整个过程保持只读，不修改 AgentJob、lease、worker、queue owner，不发送 QQ/Telegram，不触发 Python AI。
