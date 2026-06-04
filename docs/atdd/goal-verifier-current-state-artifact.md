# ATDD: Unified Goal Verifier Current-State Artifact

## Scope

- Operator-visible behavior of the unified migration verifier artifact.

## Preconditions

- Local Go runtime is reachable at `http://127.0.0.1:8780`.
- Dashboard is reachable at `http://127.0.0.1:2236`.
- Repo root is `E:\agent\my-akashic_agent`.

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`.
- Expect:
  The script prints JSON and rewrites `.codex-goal-verifier.json` in the repo
  root with the same current-turn result.

### Scenario 2

- Action:
  Open the written `.codex-goal-verifier.json`.
- Expect:
  The JSON contains top-level `current_state` with normalized fields for
  `qq_group_send_enabled`, `telegram_token_configured`, execution owners,
  `agent_job_external_lease_*`, Telegram receiver count, worker stale totals,
  and `dashboard_fallback_category`.

### Scenario 3

- Action:
  Inspect `residual_classification` and `dashboard_read_models` in the same
  artifact after the run.
- Expect:
  Existing live verifier evidence remains present, including worker-status
  cleanup, dashboard control-audit parity, QQ cutover route-matrix table, and
  control-audit / control-mutation-policy table booleans.

## Failure Signals

- `.codex-goal-verifier.json` is missing or stale after a successful run.
- `current_state` is missing or lacks the normalized live fields.
- The artifact regresses existing residual classification or dashboard
  read-model evidence.

## Evidence

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `.codex-goal-verifier.json`
