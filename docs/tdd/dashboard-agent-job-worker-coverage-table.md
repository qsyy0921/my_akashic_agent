# Dashboard Agent Job Worker Coverage Table TDD

## Tests

1. `tests/test_runtime_overview_dashboard_plugin.py`
   - 断言 panel JS 资产包含：
     - `Agent Job Worker Coverage`
     - `No worker coverage sampled`
     - `Expected Workers`
2. `scripts/verify-go-migration-goal.ps1`
   - `dashboard_read_models.agent_job_worker_coverage_table=true`
   - `residual_classification.dashboard_fallback.status=live_verified`
