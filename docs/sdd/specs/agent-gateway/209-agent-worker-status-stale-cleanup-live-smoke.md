# 209 Agent Worker Status Stale Cleanup Live Smoke

## Context

Go `agent-runtime` 已经拥有 `agent-worker-statuses` 的 report、list、restart
takeover、lease fencing 和 heartbeat renewal 语义。

但在长期运行 runtime 里，超过 stale threshold 的旧 heartbeat 记录此前只能在
`GET /v1/agent-worker-statuses` 时被只读降级成 `stale=true/status=stopped`，
没有显式 cleanup mutation 去删除这些已经失效的 runtime-state 记录。

这会留下两个问题：

- operator 能看见 stale worker，却不能安全清掉它；
- runtime overview 的 `Agent Workers` card 会长期带着历史 stale 残留。

## Decision

新增 Go-owned stale cleanup mutation：

- `POST /v1/agent-worker-statuses/cleanup-stale`

并新增 repo-owned verifier：

- `scripts/verify_agent_worker_status_cleanup_live_smoke.py`
- `scripts/verify-agent-worker-status-cleanup-live-smoke.ps1`

## API

请求体可为空，也可带过滤条件：

```json
{
  "worker_id": "akashic-python-worker",
  "instance_id": "akashic-python-worker:5776:ddd97a0a6e14",
  "timestamp": "2026-06-03T06:20:00Z",
  "stale_after_seconds": 180
}
```

返回：

```json
{
  "deleted": [...],
  "remaining": [...],
  "totals": {
    "deleted": 1,
    "remaining": 3,
    "remaining_stale": 0
  },
  "notes": [
    "side_effect=runtime_state_only",
    "stale_agent_worker_statuses_removed"
  ],
  "side_effect": "runtime_state_only"
}
```

## Requirements

- 只允许删除超过 stale threshold 的 agent worker status 记录。
- 已显式 `failed/stopped` 且不是 stale-heartbeat 推导出来的记录，不应被这个
  cleanup 自动删除。
- `worker_id` / `instance_id` 过滤是可选的；提供后只清理匹配项。
- 删除时必须同时更新：
  - in-memory service state
  - `agent-worker-statuses.json`
- 返回中必须给出 `deleted/remaining/totals/notes/side_effect`。
- `side_effect` 固定为 `runtime_state_only`。

## Non-Goals

- 不启动、停止或重启 Python worker。
- 不修改 AgentJob、QQ/Telegram、scheduler、OCR/VLM、RAG 或任何 AI 执行状态。
- 不把 autoscaling / priority executor 一起做掉。

## Verification

- `uv run pytest tests/test_verify_agent_worker_status_cleanup_live_smoke.py -q`
- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./app/service ./trigger/http ./infrastructure/agentworkerstatusstore`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-cleanup-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-cleanup-live-smoke.ps1 -ApplyLiveRuntimeCleanup`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
