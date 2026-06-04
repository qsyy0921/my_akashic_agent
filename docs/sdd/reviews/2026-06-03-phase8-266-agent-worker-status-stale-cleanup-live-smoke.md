# Review: Agent Worker Status Stale Cleanup Live Smoke

## Scope

- 新增 Go-owned stale worker cleanup mutation
- 新增 repo-owned temp-runtime + live-runtime cleanup verifier
- 将 cleanup verifier 接入 unified goal verifier

## What Changed

- 新增 `POST /v1/agent-worker-statuses/cleanup-stale`
- `AgentWorkerStatusService` 现已支持删除 stale heartbeat 残留记录
- `agent-worker-statuses.json` 现已支持持久化删除
- 新增 `scripts/verify_agent_worker_status_cleanup_live_smoke.py`
- 新增 `scripts/verify-agent-worker-status-cleanup-live-smoke.ps1`
- 新增 `tests/test_verify_agent_worker_status_cleanup_live_smoke.py`
- `scripts/verify-go-migration-goal.ps1` 现已透出
  `runtime_invariants.agent_worker_status_cleanup`

## Checks

- cleanup 只删除 stale worker，不误删活跃 worker
- temp runtime 的 API 视图和 state file 都反映删除结果
- 当前 live runtime 的历史 stale worker 已可通过新 endpoint 清理
- cleanup 后 live `/v1/agent-worker-statuses.totals.stale=0`
- cleanup 后 live `/v1/runtime-overview.summary.agent_workers_stale=0`

## Residual Risk

- 这条 cleanup 只处理 stale heartbeat 残留，不提供 autoscaling / concurrency /
  priority executor
- 如果未来需要删除 operator 显式停掉的历史 worker 记录，应单独设计新的
  retention / archive 语义，不应复用 stale cleanup
