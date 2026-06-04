# 174 Dashboard Scheduler Jobs Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `scheduler_jobs` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 scheduler 当前是否空闲、trigger/tier 分布、due-soon 窗口以及最近 job 样本。

## Decision

Python dashboard panel 为 `scheduler_jobs` card 增加结构化只读 drilldown，直接展示：

- `sampled/enabled/disabled/overdue/due_soon/instant/soft/due_soon_seconds`
- `jobs_by_trigger/jobs_by_tier/jobs_by_status`
- recent sample jobs 的 `job_id/trigger/tier/status/channel/next_run/reason`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、修改或删除 scheduler job
- 不执行 complete/recovery/reconcile
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
