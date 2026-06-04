# 206 Agent Job External Lease Cutover Preflight Live Verifier

## Context

Go 侧已经有：

- `agent_job` external lease readiness
- `agent_job` external lease plan
- repo-owned temp NATS result-ack smoke

但还缺一条 repo-owned isolated preflight verifier，把下面两件事同时固定下来：

1. **没有 external-lease / result-ack flags 时，当前 owner/readiness 为什么仍 blocked；**
2. **当 temp runtime 满足 NATS、explicit flags、worker coverage 三组前提后，`agent_job_execution_owner`、queue topology ack owner、readiness 和 plan 会一起翻到 ready。**

这条验证属于 Go control plane / cutover preflight，不涉及真实生产 runtime 切换，也不涉及 Python AI 推理本身。

## Decision

新增隔离 verifier：

- `scripts/verify_agent_job_external_lease_cutover_preflight.py`
- `scripts/verify-agent-job-external-lease-cutover-preflight.ps1`

该 verifier 必须跑两个 temp runtime 场景：

1. `config_blocked`
   - 不注入 external-lease / result-ack flags
   - 读取 `/v1/queue-backend`、`/v1/queue-topology`、`/v1/agent-job-external-lease/readiness`、`/v1/agent-job-external-lease/plan`
   - 固定当前 blocked 语义和 state-store owner
2. `preflight_ready`
   - 启动 temp NATS
   - 以 full cutover flags 启动 temp runtime
   - 创建一个 old pending `group_memory_extract` job，先让 worker coverage 落到 danger
   - 再上报一个 active `knowledge` worker heartbeat
   - 验证 owner / queue topology / readiness / plan 从 worker-blocked 翻到 ready

## Requirements

- `config_blocked` 场景必须证明：
  - `queue_backend.provider=local`
  - `agent_job_execution_owner=python_ai_worker_state_store_lease`
  - readiness 含 `external_lease_not_configured`
  - readiness 含 `strict_lease_token_disabled`
- `preflight_ready` 场景必须证明：
  - `queue_backend.provider=nats_jetstream`
  - base external lease `allow_execution=true`
  - `agent_job_execution_owner=python_ai_worker_with_nats_result_ack`
  - queue topology `agent_job.ack_owner=nats_external_lease_result_ack`
  - 在没有 active worker 时，readiness 因 `agent_job_worker_coverage_blocked` 保持 not ready
  - 上报 active `knowledge` worker 后，readiness 变为 `ready=true`
  - 上报 active worker 后，plan 变为 `ready=true` 且 `decision=ready`
  - runtime overview summary 同步显示 ready owner
- verifier 全程隔离，不修改长期运行的 `127.0.0.1:8780`

## Non-Goals

- 不切换生产 runtime 到 external lease
- 不修改任何长期运行环境变量
- 不执行真实 QQ / Telegram 发送
- 不执行真实模型、RAG、OCR、VLM 或图片生成

## Verification

- `uv run pytest tests/test_verify_agent_job_external_lease_cutover_preflight.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-preflight.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
