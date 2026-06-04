# ATDD: Dashboard Proactive Tick Logs Runtime Fallback Live Verifier

## Scope

- 验证 dashboard proactive tick-log list/detail/steps 在 SQLite mirror 为空时，
  能从 Go runtime fallback 读取当前 tick state。

## Preconditions

- 本机可启动隔离 temp `agent-runtime`
- Python/FastAPI dashboard app 可在本地 TestClient 中启动
- 不需要 QQ、Telegram、LLM 或外部凭证

## Scenarios

### Scenario 1

- Action:
  启动 temp runtime，写入一条 runtime tick log；在空 SQLite dashboard
  workspace 上请求 `/api/dashboard/proactive/tick_logs`、
  `/api/dashboard/proactive/tick_logs/{tick_id}`、
  `/api/dashboard/proactive/tick_logs/{tick_id}/steps`。
- Expect:
  三个 dashboard endpoint 都返回 runtime tick 数据。

### Scenario 2

- Action:
  在 dashboard fallback 读取前后检查 workspace `proactive.db` 的 `tick_log`
  行数。
- Expect:
  前后都保持 0；dashboard 不把 runtime tick 数据写回 SQLite。

## Failure Signals

- dashboard list/detail/steps 任一返回空或 404
- SQLite `tick_log` 在 fallback 读取后出现新行
- verifier 无法启动 temp runtime

## Evidence

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-proactive-tick-logs-boundary.ps1`
- 输出中的 `sqlite_tick_logs_before/after`
- 输出中的 `dashboard_tick_log_list/detail/steps`
