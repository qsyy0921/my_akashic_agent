# 198 Scheduler Runtime Mutation Live Smoke

Date: 2026-06-03

## Background

`036` through `041` already moved scheduler snapshot, execution lease,
single-job CRUD, completion mutation, and startup recovery reconciliation into
the Go/Python boundary described for this repo.

The remaining gap was not missing code. It was missing current-turn runtime
evidence that these write paths and reconciliation paths still behave correctly
without touching the long-running local runtime on `127.0.0.1:8780`.

## Scope

Add a dedicated scheduler live-smoke verifier that:

- starts an isolated temporary `agent-runtime` process with its own
  `AKASHIC_RUNTIME_STATE_DIR`;
- verifies `POST /v1/scheduler/jobs/upsert` and
  `DELETE /v1/scheduler/jobs/{job_id}` persist to `scheduler-jobs.json`;
- verifies lease-guarded
  `POST /v1/scheduler/jobs/{job_id}/complete` for:
  - one-shot delete;
  - recurring reschedule;
- verifies Python `SchedulerService.load_and_recover()` against the isolated Go
  runtime:
  - advances overdue recurring jobs;
  - deletes expired one-shot jobs;
  - preserves future jobs;
  - does not require QQ/Telegram delivery;
- exposes the result through the unified goal verifier.

## Non-Goals

- Do not mutate the long-running local runtime on `127.0.0.1:8780`.
- Do not use QQ/Telegram adapters or send platform messages.
- Do not move scheduler tick, cron/interval semantics, or AI execution into Go.
- Do not turn schedule-tool UX smoke into a hard blocker for Go migration.

## Design

Implementation adds:

- `scripts/verify_scheduler_runtime_live_smoke.py`
- `scripts/verify-scheduler-runtime-live-smoke.ps1`

The Python verifier:

1. launches `go run ./cmd/agent-runtime` on a free local port;
2. points it at a temporary runtime state directory;
3. drives synthetic scheduler jobs through Go HTTP endpoints;
4. reads the isolated `scheduler-jobs.json`;
5. runs Python recovery logic against that isolated runtime only;
6. emits JSON evidence for the unified verifier.

## Acceptance

- Unified verifier reports
  `residual_classification.scheduler.category=go_control_plane_mutation_and_recovery_live_verified`
  when all isolated smoke checks pass.
- The verifier output includes separate evidence for:
  - CRUD persistence;
  - completion mutation;
  - recovery reconciliation.
- Failure in any sub-check downgrades the scheduler verifier conclusion instead
  of silently claiming success.

## Evidence

- `uv run pytest tests/test_verify_scheduler_runtime_live_smoke.py -q`
- `uv run python scripts/verify_scheduler_runtime_live_smoke.py --repo-root <repo>`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
