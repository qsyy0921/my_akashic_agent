# Dashboard Receiver Leases Table TDD

## Tests

1. `tests/test_runtime_overview_dashboard_plugin.py`
   - 断言 panel JS 资产包含：
     - `Receiver Leases`
     - `No receiver leases sampled`
     - `Owner Instance`
2. `scripts/verify-go-migration-goal.ps1`
   - `dashboard_read_models.receiver_leases_table=true`
   - `residual_classification.dashboard_fallback.status=live_verified`
