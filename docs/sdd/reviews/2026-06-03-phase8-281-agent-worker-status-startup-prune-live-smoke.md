# 2026-06-03 Phase8-281 Agent Worker Status Startup Prune Live Smoke

## 结论

已完成。

## 本轮交付

- Go runtime 现在会在 repository load 时自动 prune stale
  `agent-worker-statuses`
- 新增 repo-owned temp-runtime startup smoke，固定 stale-on-restart 不再回灌
- unified goal verifier 的 `current_state.agent_workers_*` 改为直接读取当前
  turn live `/v1/agent-worker-statuses`
- 当前 8780 live runtime 已重启并吃到这轮代码，worker stale totals 保持为 0

## 已验证

- `go test ./app/service ./trigger/http ./infrastructure/agentworkerstatusstore`
- `uv run pytest tests/test_verify_agent_worker_status_startup_prune_live_smoke.py tests/test_verify_go_migration_goal_script.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-startup-prune-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`

## 当前 live 结果

- `/v1/agent-worker-statuses.totals.workers=0`
- `/v1/agent-worker-statuses.totals.stale=0`
- `/v1/runtime-overview.summary.agent_workers_stale=0`
- `knowledge_job_planner_readiness_worker_stale=0`
- unified artifact:
  - `current_state.agent_workers_total=0`
  - `current_state.agent_workers_stale=0`
  - `current_state.agent_workers_stopped=0`

## 风险边界

- 本轮只修正 stale worker heartbeat 残留在 runtime restart 后重新载入的问题
- 不启动 Python worker，不恢复 knowledge backlog，不改变 AgentJob/AI ownership
