# TDD - Dashboard Media Asset Retention Plan Cleanup Table

## Tests

- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言 panel JS 资产包含：
  - `Media Asset Retention Plan`
  - `No required steps sampled`
  - `Verification Steps`
  - `Rollback Steps`
  - `Media Asset Retention Cleanup`
  - `No media asset retention cleanup notes`
  - `Candidates`
  - `Rolled Back`
- 继续运行 SDD governance/spec index 测试，确保新 spec 已被索引。
