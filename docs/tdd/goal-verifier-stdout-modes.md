# Goal Verifier Stdout Modes TDD

## 新增测试

- `tests/test_verify_go_migration_goal_script.py`
  - `StdoutMode` 参数声明存在且默认值为 `summary`
  - `none/full/summary` 三种分支都存在
  - artifact 仍默认写入 `.codex-goal-verifier.json`
  - summary stdout 仍透出 `artifact_path/current_state/migration_bucket_summary`

## 回归检查

- `current_state` 字段集合不回退
- `migration_residuals` 与 `migration_bucket_summary` 不回退
- `dashboard_read_models` 与 `checks.dashboard_read_models` 双入口不回退
- 不更改任何 live verifier wiring
