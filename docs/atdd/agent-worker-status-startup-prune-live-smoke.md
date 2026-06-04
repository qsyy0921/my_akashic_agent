# Agent Worker Status Startup Prune ATDD

## 场景

当 Go runtime 从 `agent-worker-statuses.json` 启动时，历史 stale heartbeat
记录不应重新进入 live runtime 视图。

## 验收步骤

1. 预写一份包含 1 条 stale worker 和 1 条 active worker 的
   `agent-worker-statuses.json`
2. 启动隔离 temp `agent-runtime`
3. 调用：
   - `GET /v1/agent-worker-statuses`
   - `GET /v1/runtime-overview`
4. 断言：
   - stale worker 不再出现在 API
   - active worker 仍保留
   - state file 里 stale worker 已被删除
   - `runtime_overview.summary.agent_workers_stale == 0`
