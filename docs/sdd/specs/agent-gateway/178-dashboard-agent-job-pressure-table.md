# 178 Dashboard Agent Job Pressure Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `agent_job_metrics.pressure` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 pending/active 压力、throughput 和 dead-letter 基线。

## Decision

Python dashboard panel 为 `agent_job_pressure` card 增加结构化只读 drilldown，直接展示：

- `job_types/high_pressure/max_pending/max_active/oldest_pending_age_seconds`
- throughput 的 `created/succeeded`
- dead letters current total
- `pressure.by_type` 的 `job_type/pending/leased/running/active/oldest_pending_age_seconds/high_pressure`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、租约或执行 AgentJob
- 不修改 queue/backpressure 参数
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
