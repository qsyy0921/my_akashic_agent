# TDD: Unified Goal Verifier Current-State Artifact

## Scope

- Protect the unified goal verifier contract that persists the current-turn JSON
  artifact and exposes a normalized `current_state` summary.

## Target Code Paths

- `scripts/verify-go-migration-goal.ps1`
- `tests/test_verify_go_migration_goal_script.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| default artifact write | script contract | verifier declares a default `.codex-goal-verifier.json` output path and writes JSON to it |
| normalized state snapshot | script contract | verifier exposes a top-level `current_state` object with the required live fields |
| no regression to existing wiring | script contract | unified verifier still references dashboard control-audit and worker-status cleanup evidence |

## Required Automated Tests

- `uv run pytest tests/test_verify_go_migration_goal_script.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Full end-to-end JSON content validation remains a live smoke concern because
  the verifier depends on current local runtime and dashboard state.
