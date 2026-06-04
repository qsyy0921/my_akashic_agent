# 204 Agent Worker Status Fencing Live Smoke

## Context

Go 已经在 `/v1/agent-worker-statuses/report` 上实现了 `worker_id + instance_id +
lease_until` 的 fencing，并且已有 unit/handler tests 与 restart-takeover live
verifier。

但当前仍缺一条 repo-owned live smoke，去证明另一个关键运行时不变量：

- **同一 `worker_id` 的第二个 live instance 默认会被 `HTTP 409` 拒绝；**
- 只有显式提供匹配当前 lease owner 的
  `replace_existing_instance_id` 时，Go 才接受受控 takeover。

这是 Go 侧 runtime invariant，不属于 Python AI 推理层。

## Decision

新增隔离 temp-runtime verifier：

- `scripts/verify_agent_worker_status_fencing_live_smoke.py`
- `scripts/verify-agent-worker-status-fencing-live-smoke.ps1`

该 verifier 必须：

1. 启动隔离 temp `agent-runtime`
2. 对同一 synthetic `worker_id`：
   - 先上报 `instance_a`
   - 再上报 `instance_b`，不带 `replace_existing_instance_id`
   - 预期收到 `HTTP 409`
   - 最后带 `replace_existing_instance_id=instance_a` 再次上报 `instance_b`
   - 预期 takeover 成功
3. 同时检查：
   - `/v1/agent-worker-statuses` 返回的 active record
   - `agent-worker-statuses.json` 的最终持久化内容

## Requirements

- 默认冲突路径必须返回 `HTTP 409`
- 冲突文本必须包含 `existing_instance_id`
- takeover 只允许在 `replace_existing_instance_id` 与当前 owner 完全匹配时成功
- takeover 后：
  - `GET /v1/agent-worker-statuses` 的该 `worker_id` 应切到 `instance_b`
  - `agent-worker-statuses.json` 也应切到 `instance_b`
- verifier 全程隔离，不操作长期运行的 `127.0.0.1:8780`

## Non-Goals

- 不验证真实 Python `main.py` restart takeover；这已由现有 verifier 覆盖
- 不验证长任务 heartbeat renewal
- 不触发 AgentJob、QQ/Telegram 发送、OCR/VLM、RAG 或任何 AI side effect

## Verification

- `uv run pytest tests/test_verify_agent_worker_status_fencing_live_smoke.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-worker-status-fencing-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
