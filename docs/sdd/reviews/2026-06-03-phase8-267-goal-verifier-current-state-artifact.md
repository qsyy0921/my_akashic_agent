# Phase 8 Review 267 - Unified Goal Verifier Current-State Artifact

## What Changed

- Added a canonical artifact write path to
  `scripts/verify-go-migration-goal.ps1`.
- Added a top-level `current_state` summary with normalized live runtime
  cutover, Telegram, agent-worker, and dashboard-fallback fields.
- Added `tests/test_verify_go_migration_goal_script.py` to lock the script
  contract.

## Why

The unified verifier already had the strongest current-turn migration evidence,
but operators still had to rely on stdout or older ad-hoc artifacts. This slice
makes the repo-local JSON artifact deterministic and stable for follow-up
classification without changing any runtime owner or cutover gate.

## Evidence

- `uv run pytest tests/test_verify_go_migration_goal_script.py tests/test_verify_dashboard_control_audit_boundary.py tests/test_verify_agent_worker_status_cleanup_live_smoke.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Residual Risk

- This slice improves current-turn evidence persistence only. It does not change
  the actual runtime blockers:
  - `telegram_token_missing`
  - `qq_image_native_platform_blocker_unresolved`
