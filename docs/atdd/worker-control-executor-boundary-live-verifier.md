# Worker Control Executor Boundary Live Verifier ATDD

## Scope

- Operator-visible proof that Go already owns the worker-control control plane
  for capacity and priority, while runtime still lacks an actual executor.

## Preconditions

- Local `agent-runtime` is reachable at `http://127.0.0.1:8780`
- `agent_job capacity plan`, `agent_job priority plan`, and
  `control mutation policy` endpoints are enabled

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\verify-worker-control-executor-boundary.ps1`
- Expect:
  It returns structured JSON proving:
  - capacity/priority plan summary matches runtime overview summary
  - priority plan is still `manual_only`
  - control mutation policy already allowlists capacity/priority
  - runtime workers do not yet expose an autoscaling / concurrency / priority
    executor
  - `conclusion.category=go_control_plane_live_verified_without_worker_control_executor`

### Scenario 2

- Action:
  Run `.\scripts\verify-go-migration-goal.ps1`
- Expect:
  The unified goal output now includes:
  - `worker_control_executors`
  - `residual_classification.worker_control_executors`
  - `residual_classification.autoscaling_executor`
  and the last two reuse the same live category instead of a static string.

## Failure Signals

- 仍需要人工拼 capacity/priority/policy/runtime-workers 才能判断边界
- unified goal verifier 继续输出写死的 `go_plan_only`
- runtime 已经出现 executor worker，但 verifier 仍把它当作 plan-only

## Evidence

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-worker-control-executor-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
