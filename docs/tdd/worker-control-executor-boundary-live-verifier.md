# Worker Control Executor Boundary Live Verifier TDD

## Scope

- Protect the repo-owned verifier that classifies capacity/priority control
  plane as plan-only or executor-present.

## Target Code Paths

- `scripts/verify_worker_control_executor_boundary.py`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| plan-only live state | unit | verifier returns `go_control_plane_live_verified_without_worker_control_executor` |
| drifted priority/executor state | unit | verifier returns `worker_control_executor_boundary_mismatch_detected` |
| unified goal wiring | integration/live | goal verifier reuses the new live conclusion for worker-control residual classification |

## Required Automated Tests

- `uv run pytest tests/test_verify_worker_control_executor_boundary.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-worker-control-executor-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- No automated test creates a real autoscaling / priority executor worker yet.
  That transition remains future live smoke after an approved executor slice
  exists.
