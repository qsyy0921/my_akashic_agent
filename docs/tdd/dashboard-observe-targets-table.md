# Dashboard Observe Targets Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Observe Targets`
  - 校验 panel 资产包含 `No observe targets sampled`
  - 校验 panel 资产包含 `No observe-target notes sampled`
  - 校验 panel 资产包含 `Reply Allowed`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.observe_targets_table`
