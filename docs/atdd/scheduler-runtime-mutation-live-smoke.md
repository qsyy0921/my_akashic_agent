# ATDD: Scheduler Runtime Mutation Live Smoke

## Scope

- Operator-visible proof that Go-owned scheduler mutation and recovery paths
  still work in a real runtime process without touching the long-running local
  runtime.

## Preconditions

- Go toolchain is available locally.
- Repo dependencies are installed well enough to run `uv run python`.
- No QQ/Telegram send is required or allowed for this acceptance path.

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\verify-scheduler-runtime-live-smoke.ps1`.
- Expect:
  The script starts an isolated temporary `agent-runtime`, writes a synthetic
  one-shot scheduler job through `upsert`, confirms the job appears in
  `scheduler-jobs.json`, then deletes it and confirms both HTTP state and file
  state are clean again.

### Scenario 2

- Action:
  In the same isolated runtime, let the script acquire a scheduler execution
  lease and call `complete` once with `action=delete` and once with
  `action=reschedule`.
- Expect:
  The one-shot job disappears, the recurring job is updated in place, and both
  paths release their leases without sending QQ/Telegram messages.

### Scenario 3

- Action:
  In the same isolated runtime, let the script seed overdue recurring,
  expired one-shot, and future one-shot jobs, then call Python
  `SchedulerService.load_and_recover()` against that isolated runtime.
- Expect:
  The recurring job advances to a future fire time, the expired one-shot job is
  deleted, the future job remains, diagnostics show no overdue jobs, and no
  platform delivery is attempted.

## Failure Signals

- Temporary runtime fails to reach `/healthz`.
- `scheduler-jobs.json` does not reflect job creation, reschedule, or deletion.
- `complete` leaves a stale lease or fails to update the intended job.
- Recovery leaves overdue jobs behind or removes unaffected future jobs.
- Any step requires QQ/Telegram delivery to succeed.

## Evidence

- JSON output from `.\scripts\verify-scheduler-runtime-live-smoke.ps1`
- JSON output from `.\scripts\verify-go-migration-goal.ps1`
- Temporary runtime stdout/stderr log paths emitted by the smoke script
