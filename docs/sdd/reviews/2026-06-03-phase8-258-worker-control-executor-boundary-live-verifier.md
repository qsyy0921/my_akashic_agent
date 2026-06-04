# Review: worker control executor boundary live verifier

Spec:
- `docs/sdd/specs/agent-gateway/201-worker-control-executor-boundary-live-verifier.md`

Implementation summary:
- Added `scripts/verify_worker_control_executor_boundary.py` and
  `scripts/verify-worker-control-executor-boundary.ps1` to read
  `agent_job capacity/priority` plans, control-mutation policy, runtime workers,
  and runtime-overview summary, then emit structured JSON for their current-turn
  parity and executor presence.
- Updated `scripts/verify-go-migration-goal.ps1` to include the verifier output
  as `worker_control_executors`, reuse its conclusion for
  `residual_classification.worker_control_executors`, and replace the previous
  static `autoscaling_executor` classification with the same live result.

Tests run:
- `uv run pytest tests/test_verify_worker_control_executor_boundary.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-worker-control-executor-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

Findings:
- Current live runtime proves the control plane exists:
  - capacity plan is ready and read-only
  - priority plan is ready, `manual_only`, and still Python-owned for execution
  - control-mutation policy already allowlists `agent_job_capacity` and
    `agent_job_priority`
- Current live runtime also proves the executor does not exist yet:
  - runtime workers expose only outbox, knowledge-planner, and NATS queue
    migration workers
  - no autoscaling, concurrency, or priority executor worker is present
- The remaining gap is therefore executor implementation and cutover discipline,
  not visibility or plan availability.

Decision:
- Accept.

Follow-ups:
- Reuse this verifier before and after any future worker-control executor slice
  so plan-only and executor-present states cannot be confused.
