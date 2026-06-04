# TDD: Agent Worker Status Stale Cleanup Live Smoke

## Scope

- 保护 stale worker cleanup 的 service、HTTP contract、store delete 和
  repo-owned live smoke。

## Target Code Paths

- `services/agent-runtime/app/service/agent_worker_status_service.go`
- `services/agent-runtime/trigger/http/handler.go`
- `services/agent-runtime/infrastructure/agentworkerstatusstore/store.go`
- `scripts/verify_agent_worker_status_cleanup_live_smoke.py`
- `scripts/verify-agent-worker-status-cleanup-live-smoke.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| cleanup stale only | unit/integration | 只删除超过 stale threshold 的 worker status |
| preserve active worker | unit/integration | 活跃 worker 不被误删 |
| persistent delete | unit | 删除后 reopen store 不再读到 stale worker |
| HTTP cleanup contract | integration | `POST /v1/agent-worker-statuses/cleanup-stale` 返回 deleted/remaining/totals |
| repo-owned smoke | unit/integration | temp runtime cleanup smoke 与 live runtime inspection 结构稳定 |

## Required Automated Tests

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./app/service ./trigger/http ./infrastructure/agentworkerstatusstore`
- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_worker_status_cleanup_live_smoke.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-cleanup-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- 真实 Python `main.py` restart takeover 继续由
  `scripts/verify-agent-worker-status-restart.ps1` 覆盖
- 真实 worker control executor 仍不在本 slice；该项继续留在 `OPEN_ISSUES.md`
