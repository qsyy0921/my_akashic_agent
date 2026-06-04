# Dashboard Queue Backend Table TDD

## Tests

1. `tests/test_runtime_overview_dashboard_plugin.py`
   - 断言 panel JS 资产包含：
     - `Queue Backend`
     - `No provider capabilities sampled`
     - `Consumer Model`
     - `Concurrent Consumers`
2. `scripts/verify-go-migration-goal.ps1`
   - `dashboard_read_models.queue_backend_table=true`
   - `residual_classification.dashboard_fallback.status=live_verified`
