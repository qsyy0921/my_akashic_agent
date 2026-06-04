# ATDD: Unified Goal Verifier Migration Residuals

## Scope

- Operator-visible residual classification emitted by the unified migration
  verifier.

## Preconditions

- Local Go runtime is reachable.
- Dashboard is reachable.
- The unified verifier can complete successfully.

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`.
- Expect:
  The generated `.codex-goal-verifier.json` contains top-level
  `migration_residuals`.

### Scenario 2

- Action:
  Inspect `migration_residuals.qq_napcat_cutover`,
  `migration_residuals.telegram_backend`, and
  `migration_residuals.agent_job_external_lease_result_ack`.
- Expect:
  They expose current category, key live facts, and a next minimal step rather
  than requiring manual reading of the entire verifier tree.

### Scenario 3

- Action:
  Inspect the remaining entries for scheduler, worker control executors, media
  recovery, dashboard read-models, and MQ adapter boundary.
- Expect:
  Each entry reflects current-turn live evidence and does not claim a cutover or
  executor exists when it does not.

## Failure Signals

- `migration_residuals` is missing.
- Any of the eight required residual entries is missing.
- An entry contradicts the live runtime snapshot or implies a cutover that did
  not happen.

## Evidence

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `.codex-goal-verifier.json`
