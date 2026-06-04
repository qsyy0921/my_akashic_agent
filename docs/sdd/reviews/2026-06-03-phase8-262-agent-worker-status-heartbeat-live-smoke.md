# Review: Agent Worker Status Heartbeat Live Smoke

## Scope

- 新增 repo-owned temp-runtime worker-status heartbeat renewal live smoke
- 将该 verifier 接入 unified goal verifier

## What Changed

- 新增 `scripts/verify_agent_worker_status_heartbeat_live_smoke.py`
- 新增 `scripts/verify-agent-worker-status-heartbeat-live-smoke.ps1`
- 新增 `tests/test_verify_agent_worker_status_heartbeat_live_smoke.py`
- `scripts/verify-go-migration-goal.ps1` 现已同时透出：
  - `runtime_invariants.agent_worker_status_heartbeat`
  - `residual_classification.agent_worker_status_heartbeat`

## Checks

- 同一实例的三次 `running` heartbeat 都返回 accepted
- `updated_at` 随 heartbeat 推进
- `lease_until` 随 heartbeat 推进
- final record 仍保持 `lease_active=true` 且 `stale=false`
- API 视图与 `agent-worker-statuses.json` 的最终 heartbeat 一致
- verifier 运行在隔离 temp runtime，不影响长期运行的本地 8780 runtime

## Residual Risk

- 这条 smoke 固定了 Go runtime 对 heartbeat renewal 的控制面语义，但不替代真实
  image / knowledge / rag_eval / outbox 长任务的人为观察
- 这条 smoke 不引入真实 Python worker 执行，因此不覆盖业务任务本身的执行正确性
