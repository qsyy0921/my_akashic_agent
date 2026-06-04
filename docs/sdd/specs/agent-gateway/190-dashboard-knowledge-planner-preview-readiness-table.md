# 190 Dashboard Knowledge Planner Preview Readiness Table

## Context

`/v1/runtime-overview` 已提供 `knowledge_job_planner_preview` 与
`knowledge_job_planner_readiness` card/detail，但 dashboard panel 之前只能靠 raw JSON
查看 preview bucket、observe-only group admission 计划，以及 planner/worker readiness。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为
`knowledge_job_planner_preview` 与 `knowledge_job_planner_readiness` 增加结构化只读
drilldown，展示：

- preview 的 `targets/skipped/total_jobs/group_memory_jobs/rag_ingest_jobs/interval`
- preview plan 的 `target/channel/jobs/first job type/observe_only/scheduler`
- readiness 的 `ready/planner enabled/planner running/worker ready/worker active/stale/failed/stopped`
- readiness 关联的 preview bucket/targets/total jobs/timestamp 与 notes

## Non-Goals

- 不创建 AgentJob
- 不切换 knowledge planner owner
- 不启动/停止 planner 或 Python knowledge worker

## Verification

- `GET /v1/runtime-overview`
- `GET /v1/knowledge-job-planner/readiness`
- `GET /v1/knowledge-job-planner/cutover-plan`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
