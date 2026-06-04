# TDD - Dashboard Runtime Health Stale Jobs Table

## Tests

- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言 panel JS 资产包含：
  - `Runtime Health`
  - `No runtime health snapshot fields`
  - `No runtime health errors`
  - `Stale Jobs`
  - `No worker stale-job diagnostics sampled`
  - `No stale jobs sampled`
  - `Latest Updated`
- 继续运行 SDD governance/spec index 测试，确保新 spec 已被索引。
