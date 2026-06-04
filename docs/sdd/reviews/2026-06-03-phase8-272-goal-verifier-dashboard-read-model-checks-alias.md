# Review: Goal Verifier Dashboard Read-Model Checks Alias

## What changed

- Added an explicit alias from `checks.dashboard_read_models` to the existing
  `$dashboardReadModelChecks` object in
  `scripts/verify-go-migration-goal.ps1`.
- Added a focused regression test in
  `tests/test_verify_go_migration_goal_script.py`.

## Why

The previous turn already proved the proactive dashboard tick-log runtime
fallback live path, but one artifact consumer still read
`checks.dashboard_read_models.proactive_tick_logs_readable`. The canonical JSON
only exposed that object at top level, so the consumer saw `null` despite valid
evidence.

## Evidence

- `uv run pytest tests/test_verify_dashboard_proactive_tick_logs_boundary.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `.codex-goal-verifier.json` now contains both top-level
  `dashboard_read_models` and `checks.dashboard_read_models`.
