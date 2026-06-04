# TDD: Scheduler Runtime Mutation Live Smoke

## Scope

- Protect the isolated verifier that proves Go-owned scheduler CRUD,
  completion mutation, and startup recovery reconciliation still behave as
  expected.

## Target Code Paths

- `scripts/verify_scheduler_runtime_live_smoke.py`
- `scripts/verify-scheduler-runtime-live-smoke.ps1`
- `agent/scheduler.py`
- `agent/config_models.py`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| CRUD smoke | unit/contract | synthetic job appears after `upsert`, persists to state file, and disappears after `delete` |
| completion delete/reschedule smoke | unit/contract | lease-acquired one-shot delete removes the job; recurring complete updates the job and releases the lease |
| recovery smoke | unit/contract | overdue recurring advances, expired one-shot is deleted, future job remains, diagnostics show no overdue jobs |

## Required Automated Tests

- `uv run pytest tests/test_verify_scheduler_runtime_live_smoke.py -q`
- `uv run python scripts/verify_scheduler_runtime_live_smoke.py --repo-root E:\agent\my-akashic_agent`

## Deferred Coverage

- Real schedule-tool UI/UX smoke on the long-running local runtime remains a
  live-check concern, not a deterministic automated test.
- Actual QQ/Telegram sends remain out of scope because the verifier must prove
  control-plane behavior without platform side effects.
