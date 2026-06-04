# ATDD - Dashboard Knowledge Planner Preview Readiness Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `knowledge_job_planner_preview`
与 `knowledge_job_planner_readiness` detail 时，应能直接看到 observe-only group
admission preview 和 planner/worker readiness，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Knowledge Planner Preview`。
2. panel 显示 `Group Memory Jobs`、`Observe Only`。
3. panel 显示 `Knowledge Planner Readiness`。
4. panel 显示 `Planner Running`、`Worker Active`。
5. unified goal verifier 返回：
   - `dashboard_read_models.knowledge_job_planner_preview_table=true`
   - `dashboard_read_models.knowledge_job_planner_readiness_table=true`
