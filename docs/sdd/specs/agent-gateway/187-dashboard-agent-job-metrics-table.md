# 187 Dashboard Agent Job Metrics Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `agent_job_metrics` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 AgentJob throughput、按类型分布、pressure 基线和 recent dead-letter 样本。

## Decision

Python dashboard panel 为 `agent_job_metrics` card 增加结构化只读 drilldown，直接展示：

- `sampled_jobs/sampled_events/job_types/created/leased/running/succeeded/dead_letters_current`
- `jobs_by_type` 表格
- `pressure.by_type` 表格
- `dead_letters.recent` 表格

## Constraints

- 只读展示，不新增任何 mutation UI
- 不修改 AgentJob、lease、worker 或 queue owner
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
