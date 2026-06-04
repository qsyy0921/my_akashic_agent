# Phase 8 Review 268 - Unified Goal Verifier Migration Residuals

## What Changed

- Added top-level `migration_residuals` to
  `scripts/verify-go-migration-goal.ps1`.
- The unified artifact now classifies the eight recurring migration residual
  areas with current category, key live facts, and next minimal step.
- Extended `tests/test_verify_go_migration_goal_script.py` to lock the residual
  contract.

## Why

The repo already had current-turn live evidence, but the final migration summary
still required manual reconstruction. This slice makes the classification
itself reproducible and machine-readable without changing runtime ownership or
cutover state.

## Evidence

- `uv run pytest tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Residual Risk

- The classification is only as strong as the underlying live verifiers. It
  does not remove the actual blockers:
  - `telegram_token_missing`
  - `qq_image_native_platform_blocker_unresolved`
