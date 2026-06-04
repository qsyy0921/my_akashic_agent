# TDD: Unified Goal Verifier Migration Buckets

## Scope

- Protect the exact four-bucket classification contract for the unified goal
  verifier artifact.

## Target Code Paths

- `scripts/verify-go-migration-goal.ps1`
- `tests/test_verify_go_migration_goal_script.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| per-entry bucket field | script contract | each residual entry exposes `migration_bucket` |
| exact bucket names | script contract | the verifier defines the four required bucket names |
| top-level summary | script contract | the artifact exposes `migration_bucket_summary` |

## Required Automated Tests

- `uv run pytest tests/test_verify_go_migration_goal_script.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Exact runtime bucket membership remains live-smoke dependent because it
  depends on the current verifier evidence.
