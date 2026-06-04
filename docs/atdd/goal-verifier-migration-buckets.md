# ATDD: Unified Goal Verifier Migration Buckets

## Scope

- Exact migration bucket reporting for the unified verifier artifact.

## Preconditions

- The unified verifier can complete against the local runtime.

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`.
- Expect:
  The artifact contains `migration_residuals.*.migration_bucket` for each
  residual entry.

### Scenario 2

- Action:
  Inspect `migration_bucket_summary`.
- Expect:
  The artifact groups residual items under the exact four buckets required by
  the migration close-out process.

### Scenario 3

- Action:
  Inspect the Python ownership entry.
- Expect:
  The artifact explicitly marks Python-owned AI surfaces as
  `explicitly_python_owned` instead of treating them as unfinished Go work.

## Failure Signals

- Any residual entry lacks `migration_bucket`.
- `migration_bucket_summary` is missing.
- Python-owned AI surfaces are absent from the artifact.

## Evidence

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `.codex-goal-verifier.json`
