# TDD: Agent Worker Status Heartbeat Live Smoke

## Scope

- 保护 worker-status heartbeat renewal live smoke 的 heartbeat 序列、时间推进和
  state-file 证据读取。

## Target Code Paths

- `scripts/verify_agent_worker_status_heartbeat_live_smoke.py`
- `scripts/verify-agent-worker-status-heartbeat-live-smoke.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| repeated running heartbeat | unit/integration | 同一实例连续 heartbeat 都被接受 |
| timestamp and lease advancement | unit/integration | `updated_at` 与 `lease_until` 单调推进 |
| state-file schema compatibility | unit | verifier 同时兼容 API 视图与持久化文件字段形状 |

## Required Automated Tests

- `uv run pytest tests/test_verify_agent_worker_status_heartbeat_live_smoke.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-heartbeat-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- 真实 Python `main.py` restart takeover 继续由
  `scripts/verify-agent-worker-status-restart.ps1` 覆盖
- 真实业务长任务的端到端执行期 heartbeat 仍可继续保留在 `LIVE_CHECKS.md`
  做人工观察，但 repo-owned renewal invariant 现在已有独立 smoke 固定
