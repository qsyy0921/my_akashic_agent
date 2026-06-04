# TDD: Goal Verifier Dashboard Knowledge/RAG Retry Hardening

## Scope

- Protect the unified verifier wiring that retries the dashboard
  knowledge/RAG boundary verifier once before downgrading dashboard fallback
  evidence.

## Target Code Paths

- `scripts/verify-go-migration-goal.ps1`
- `tests/test_verify_go_migration_goal_script.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| retry branch present | script-structure | unified verifier retries once when first knowledge/RAG result is not `live_verified` |
| live result adoption | script-structure | retried `live_verified` result replaces the initial failed sample |
| no artifact shape drift | script-structure | existing `current_state` and `dashboard_read_models` wiring remains intact |

## Required Automated Tests

- `uv run pytest tests/test_verify_go_migration_goal_script.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real transient dashboard timing is covered by ATDD/live verifier reruns, not
  by deterministic unit tests.
