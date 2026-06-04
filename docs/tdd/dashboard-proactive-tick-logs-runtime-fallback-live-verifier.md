# TDD: Dashboard Proactive Tick Logs Runtime Fallback Live Verifier

## Scope

- 保护 dashboard proactive tick-log runtime fallback verifier 的结果归约与
  SQLite 空态判断。

## Target Code Paths

- `scripts/verify_dashboard_proactive_tick_logs_boundary.py`
- `scripts/verify-dashboard-proactive-tick-logs-boundary.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| missing db | unit | `_count_tick_logs` 返回 0 |
| existing rows | unit | `_count_tick_logs` 返回真实行数 |
| all checks pass | unit | `_build_result` 给出 `live_verified` |
| any check fails | unit | `_build_result` 给出 `verification_incomplete` |
| unified wiring | contract | goal verifier 包含新脚本与输出字段 |

## Required Automated Tests

- `uv run pytest tests/test_verify_dashboard_proactive_tick_logs_boundary.py tests/test_verify_go_migration_goal_script.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`

## Deferred Coverage

- dashboard fallback 的真实 HTTP app + temp runtime 贯通验证留给
  `scripts/verify-dashboard-proactive-tick-logs-boundary.ps1`
