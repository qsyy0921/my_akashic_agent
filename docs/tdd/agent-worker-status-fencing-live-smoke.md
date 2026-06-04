# TDD: Agent Worker Status Fencing Live Smoke

## Scope

- 保护 worker-status fencing live smoke 的 payload、冲突检查和 state-file 证据读取。

## Target Code Paths

- `scripts/verify_agent_worker_status_fencing_live_smoke.py`
- `scripts/verify-agent-worker-status-fencing-live-smoke.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| synthetic worker fencing | unit/integration | 第二实例默认得到 `HTTP 409` |
| matching replace takeover | unit/integration | 显式 `replace_existing_instance_id` 后切到新实例 |
| state-file schema compatibility | unit | verifier 同时兼容 API 视图与持久化文件中的字段形状 |

## Required Automated Tests

- `uv run pytest tests/test_verify_agent_worker_status_fencing_live_smoke.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-fencing-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- 真实 `main.py` restart takeover 继续由
  `scripts/verify-agent-worker-status-restart.ps1` 覆盖
- 长任务 heartbeat renewal 继续留在 `LIVE_CHECKS.md`
