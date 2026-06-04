# Review: 198 Scheduler Runtime Mutation Live Smoke

Date: 2026-06-03

## Summary

- Added an isolated temp-runtime scheduler smoke verifier instead of mutating
  the long-running local runtime.
- Verified three production-facing control-plane paths with real Go HTTP state:
  - single-job CRUD persistence;
  - lease-fenced completion mutation;
  - Python startup recovery reconciliation against Go-owned scheduler state.

## What Changed

- Added `scripts/verify_scheduler_runtime_live_smoke.py`.
- Added `scripts/verify-scheduler-runtime-live-smoke.ps1`.
- Integrated the new smoke into `scripts/verify-go-migration-goal.ps1`.
- Added `tests/test_verify_scheduler_runtime_live_smoke.py`.

## Acceptance Notes

- The isolated verifier starts a real temporary `agent-runtime`.
- It writes only synthetic scheduler state under a temporary
  `AKASHIC_RUNTIME_STATE_DIR`.
- It does not use QQ/Telegram delivery as part of the verification path.

## Risks Reviewed

- Avoids corrupting current `127.0.0.1:8780` runtime state during verification.
- Keeps scheduler migration claims narrow: this proves runtime mutation and
  recovery control plane, not operator UX polish.
- Fails closed: if CRUD, completion, or recovery evidence is missing, the
  unified verifier does not report scheduler success.
