# Dashboard Runtime Workers Table TDD

## Tests

1. `tests/test_runtime_overview_dashboard_plugin.py`
   - 断言 panel JS 资产包含：
     - `Runtime Workers`
     - `No runtime workers sampled`
     - `Batch / Max In Flight`
2. `scripts/verify-go-migration-goal.ps1`
   - `dashboard_read_models.runtime_workers_table=true`
   - `residual_classification.dashboard_fallback.status=live_verified`
