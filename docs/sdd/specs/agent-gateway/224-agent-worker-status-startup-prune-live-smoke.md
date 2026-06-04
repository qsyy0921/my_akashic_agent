# 224. Agent Worker Status Startup Prune Live Smoke

## 背景

当前 Go runtime 已有 `POST /v1/agent-worker-statuses/cleanup-stale`，但如果
`agent-worker-statuses.json` 在 runtime 重启前已经留下 stale heartbeat 记录，
旧记录仍会在 service repository load 时重新进入内存，导致：

- `/v1/agent-worker-statuses` 出现过期 Python worker 残留
- `/v1/runtime-overview.summary.agent_workers_stale` 被虚高
- `knowledge_job_planner_readiness` / `agent_job_capacity_plan` /
  `agent_job_priority_plan` 被 stale worker 假阻塞

## 目标

1. Go runtime 在 repository load 时自动 prune stale worker heartbeat 残留
2. startup prune 不删除 active worker record
3. 新增 repo-owned temp-runtime live smoke，固定这条启动语义
4. unified goal verifier 纳入 `agent_worker_status_startup_prune` current-turn 证据

## 非目标

- 不启动或停止 Python worker
- 不创建、租约、取消或执行 AgentJob
- 不改变 Python AI runtime、RAG、OCR/VLM、tool execution 边界
- 不把 stale cleanup 变成 read path 的隐式副作用

## 设计

### 1. Repository load 时直接 prune stale records

`NewAgentWorkerStatusServiceWithRepository()` 在读取 repository items 后：

- 对每条记录执行 `WithStaleHeartbeat(now, staleAfter)` 投影
- 若已 stale，则直接调用 repository delete，并且不载入 service memory
- 若仍 active，则保持现有 load 语义

这样 runtime restart 不会再次把历史 stale heartbeat 带回内存。

### 2. Temp-runtime startup smoke

新增 `scripts/verify-agent-worker-status-startup-prune-live-smoke.ps1`：

1. 预写一份包含 stale + active worker 的 `agent-worker-statuses.json`
2. 启动隔离 temp `agent-runtime`
3. 读取：
   - `GET /v1/agent-worker-statuses`
   - `GET /v1/runtime-overview`
   - seed state file
4. 断言：
   - stale worker 不再出现在 API
   - stale worker 不再出现在 state file
   - active worker 仍保留
   - `runtime_overview.summary.agent_workers_stale == 0`

### 3. Unified verifier current-state 修正

`verify-go-migration-goal.ps1` 的 `current_state.agent_workers_*` 应直接来源于
当前 turn 的 live `GET /v1/agent-worker-statuses`，而不是 cleanup verifier 的
历史 before snapshot。

## 验收

1. `go test ./app/service ./trigger/http ./infrastructure/agentworkerstatusstore`
   通过
2. `uv run pytest tests/test_verify_agent_worker_status_startup_prune_live_smoke.py tests/test_verify_go_migration_goal_script.py -q`
   通过
3. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-startup-prune-live-smoke.ps1`
   返回 `conclusion.status=live_verified`
4. 重启 8780 live runtime 后再次读取：
   - `/v1/agent-worker-statuses.totals.stale == 0`
   - `/v1/runtime-overview.summary.agent_workers_stale == 0`
5. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`
   返回：
   - `current_state.agent_workers_total=0`
   - `current_state.agent_workers_stale=0`
   - `current_state.agent_workers_stopped=0`
