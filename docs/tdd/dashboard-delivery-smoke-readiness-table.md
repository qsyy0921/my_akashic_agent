# Dashboard Delivery Smoke Readiness Table TDD

## Tests

1. `tests/test_runtime_overview_dashboard_plugin.py`
   - 断言 panel JS 资产包含：
     - `Delivery Smoke Readiness`
     - `No smoke cases sampled`
     - `Chat ID`
2. `scripts/verify-go-migration-goal.ps1`
   - `dashboard_read_models.delivery_smoke_readiness_table=true`
   - `residual_classification.dashboard_fallback.status=live_verified`
