# ATDD: Agent Job External Lease Cutover Preflight Live Verifier

## Given

- 仓库内可启动隔离 temp `agent-runtime`
- 本机可启动 temp NATS
- 生产 `127.0.0.1:8780` runtime 不应被修改

## When

- 运行 `scripts/verify-agent-job-external-lease-cutover-preflight.ps1`

## Then

- `config_blocked` 场景返回：
  - local provider
  - state-store owner
  - readiness 含 `external_lease_not_configured`
  - readiness 含 `strict_lease_token_disabled`
- `preflight_ready` 场景返回：
  - `nats_jetstream/external_lease`
  - `agent_job_execution_owner=python_ai_worker_with_nats_result_ack`
  - `agent_job` queue topology ack owner 为 `nats_external_lease_result_ack`
  - old pending `group_memory_extract` job 在无 worker 时先触发 `agent_job_worker_coverage_blocked`
  - 上报 active `knowledge` worker 后，readiness 变为 ready
  - 上报 active `knowledge` worker 后，plan 变为 `ready/decision=ready`
- unified goal verifier 能透出：
  - `agent_job_external_lease_cutover_preflight`
  - `residual_classification.agent_job_external_lease_cutover_preflight`
