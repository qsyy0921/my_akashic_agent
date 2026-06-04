# TDD: Unified Goal Verifier Migration Residuals

## Scope

- Protect the script contract that exposes machine-readable residual
  classifications for the remaining Go migration areas.

## Target Code Paths

- `scripts/verify-go-migration-goal.ps1`
- `tests/test_verify_go_migration_goal_script.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| residual object present | script contract | verifier emits top-level `migration_residuals` |
| eight required entries present | script contract | verifier defines the required residual sections |
| no manual-only classification | script contract | residual entries are wired into the JSON artifact and not kept as external prose only |

## Required Automated Tests

- `uv run pytest tests/test_verify_go_migration_goal_script.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Exact residual field values remain live-smoke territory because they depend on
  the current local runtime, dashboard, and verifier outputs.
