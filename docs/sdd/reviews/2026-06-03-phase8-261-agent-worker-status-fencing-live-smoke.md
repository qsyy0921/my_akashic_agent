# Review: Agent Worker Status Fencing Live Smoke

## Scope

- 新增 repo-owned temp-runtime worker-status fencing live smoke
- 将该 verifier 接入 unified goal verifier

## What Changed

- 新增 `scripts/verify_agent_worker_status_fencing_live_smoke.py`
- 新增 `scripts/verify-agent-worker-status-fencing-live-smoke.ps1`
- 新增 `tests/test_verify_agent_worker_status_fencing_live_smoke.py`
- `scripts/verify-go-migration-goal.ps1` 现已同时透出：
  - `runtime_invariants.agent_worker_status_restart`
  - `runtime_invariants.agent_worker_status_fencing`

## Checks

- 第二实例默认返回 `HTTP 409`
- 冲突文本包含 `existing_instance_id`
- 只有匹配 `replace_existing_instance_id` 时 takeover 才成功
- API 视图与 `agent-worker-statuses.json` 的最终 owner 一致
- verifier 运行在隔离 temp runtime，不影响长期运行的本地 8780 runtime

## Residual Risk

- 这条 smoke 证明了默认 fencing 与显式 takeover 边界，但没有覆盖长任务 heartbeat
  renewal；该项仍需保留在 `LIVE_CHECKS.md`
- 这条 smoke 不触发真实 Python worker，因此不证明业务任务执行期的连续 heartbeat
  续租，只证明 Go 控制面的 fencing 语义
