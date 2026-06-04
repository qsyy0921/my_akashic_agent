# 205 Agent Worker Status Heartbeat Live Smoke

## Context

Go `agent-worker-statuses` 已经有：

- worker-status registry
- lease / fencing
- restart takeover
- Python 侧 heartbeat renewal 代码路径

但当前还缺一条 repo-owned live smoke，把下面这个运行时不变量固定下来：

- **同一 `worker_id + instance_id` 在长任务期间重复上报 `running` heartbeat 时，
  Go 会持续刷新 `updated_at` 与 `lease_until`；**
- **同一实例的 heartbeat 不会被误判为 lease conflict；**
- **最终 API 视图和 `agent-worker-statuses.json` 会反映最新 heartbeat。**

这条验证属于 Go control plane / runtime invariant，不涉及 Python AI 推理与算法层。

## Decision

新增隔离 temp-runtime verifier：

- `scripts/verify_agent_worker_status_heartbeat_live_smoke.py`
- `scripts/verify-agent-worker-status-heartbeat-live-smoke.ps1`

该 verifier 必须：

1. 启动隔离 temp `agent-runtime`
2. 对同一 synthetic `worker_id` 和同一 `instance_id`：
   - 连续上报至少三次 `running` heartbeat
   - 每次 heartbeat 使用递增 timestamp
3. 同时检查：
   - `/v1/agent-worker-statuses` 的该 worker record
   - `agent-worker-statuses.json` 的最终持久化内容

## Requirements

- 三次 heartbeat 都必须被接受
- `updated_at` 必须随 heartbeat 单调推进
- `lease_until` 必须随 heartbeat 单调推进
- final record 必须保持：
  - `lease_active=true`
  - `stale=false`
- `agent-worker-statuses.json` 必须与最终 API 视图一致
- verifier 全程隔离，不操作长期运行的 `127.0.0.1:8780`

## Non-Goals

- 不验证 stale-instance takeover；这已由现有 fencing / restart smoke 覆盖
- 不触发真实 image / knowledge / rag_eval / outbox 业务任务
- 不触发 AgentJob、QQ/Telegram 发送、OCR/VLM、RAG 或任何 AI side effect

## Verification

- `uv run pytest tests/test_verify_agent_worker_status_heartbeat_live_smoke.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-heartbeat-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
