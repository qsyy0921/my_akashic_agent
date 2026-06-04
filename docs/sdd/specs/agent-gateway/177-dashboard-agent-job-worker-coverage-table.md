# 177 Dashboard Agent Job Worker Coverage Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `agent_job_worker_coverage` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查每种 AgentJob 是否有预期 worker 覆盖、是否存在 stale/failed worker、以及高压下 coverage 是否仍成立。

## Decision

Python dashboard panel 为 `agent_job_worker_coverage` card 增加结构化只读 drilldown，直接展示：

- `job_type/expected_worker_types/coverage_status/worker_count/active/running/failed/stale/high_pressure/reason`

## Constraints

- 只读展示，不新增任何 mutation UI
- 不启动、停止或替换 Python worker
- 不创建、租约或执行 AgentJob
- 不发送 QQ/Telegram
- 不触发 Python AI

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
