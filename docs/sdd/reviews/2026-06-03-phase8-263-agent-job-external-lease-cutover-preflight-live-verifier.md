# Review: Agent Job External Lease Cutover Preflight Live Verifier

## Scope

- 新增 repo-owned isolated `agent_job external lease result-ack` cutover preflight verifier
- 将该 verifier 接入 unified goal verifier

## What Changed

- 新增 `scripts/verify_agent_job_external_lease_cutover_preflight.py`
- 新增 `scripts/verify-agent-job-external-lease-cutover-preflight.ps1`
- 新增 `tests/test_verify_agent_job_external_lease_cutover_preflight.py`
- `scripts/verify-go-migration-goal.ps1` 现已同时透出：
  - `agent_job_external_lease_cutover_preflight`
  - `residual_classification.agent_job_external_lease_cutover_preflight`

## Checks

- `config_blocked` 场景固定了当前 blocked 语义：
  - local provider
  - state-store owner
  - readiness 含 `external_lease_not_configured`
  - readiness 含 `strict_lease_token_disabled`
- `preflight_ready` 场景固定了 cutover preflight 语义：
  - temp NATS + full result-ack flags 后，owner 提升为 `python_ai_worker_with_nats_result_ack`
  - queue topology `agent_job.ack_owner` 切到 `nats_external_lease_result_ack`
  - old pending `group_memory_extract` 在无 worker 时先触发 `agent_job_worker_coverage_blocked`
  - active `knowledge` worker heartbeat 上报后，readiness 与 plan 都翻到 ready

## Residual Risk

- 这条 smoke 固定了 isolated preflight 语义，但不等于生产 runtime 已切换到 external lease
- 生产 cutover 仍依赖真实 NATS backend、明确 flag 注入意图和 owner 切换窗口
- 这条 smoke 不执行真实 AI side effects，也不验证 QQ / Telegram 发送链路
