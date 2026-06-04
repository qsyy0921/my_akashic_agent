# TDD - Dashboard Dead Letters Table

## Tests

- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言 panel JS 资产包含：
  - `Dead Letters`
  - `No agent-job dead letters sampled`
  - `No outbox dead letters sampled`
  - `Agent Job Dead Letter Totals`
- 继续运行 SDD governance/spec index 测试，确保新 spec 已被索引。
