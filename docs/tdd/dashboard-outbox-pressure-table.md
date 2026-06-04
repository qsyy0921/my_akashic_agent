# Dashboard Outbox Pressure Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - fixture payload 增加 `outbox_pressure`
  - 校验 runtime overview 聚合结果包含 `outbox_pressure`
  - 校验 panel 资产包含 `Outbox Pressure`、`No outbox pressure sampled`、`Dead Lettered`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.outbox_pressure_table`
