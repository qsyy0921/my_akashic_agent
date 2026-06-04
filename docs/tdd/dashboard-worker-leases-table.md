# TDD - Dashboard Worker Leases Table

## Tests

- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言：
  - runtime overview payload 中存在 `worker_leases` card
  - old-payload fallback case 的 `value/status` 正确
- 断言 panel JS 资产包含：
  - `Worker Leases`
  - `Checkpoint Prefix`
  - `Stale Leases`
  - `No worker lease diagnostics sampled`
- 继续运行 SDD governance/spec index 测试，确保新 spec 已被索引。
