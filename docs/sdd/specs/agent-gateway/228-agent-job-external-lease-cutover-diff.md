# 228 Agent Job External Lease Cutover Diff

## Context

`agent_job external lease result-ack` 已经具备：

- `GET /v1/agent-job-external-lease/readiness`
- `GET /v1/agent-job-external-lease/plan`
- `GET /v1/agent-job-external-lease/preflight`
- `GET /v1/agent-job-external-lease/launcher-bundle`
- repo-owned temp NATS smoke
- repo-owned isolated cutover preflight verifier
- repo-owned launcher preflight / launcher bundle verifier

但当前 production runtime 还没有一条 Go-owned 只读接口，把“live runtime 与
canonical launcher bundle 的差异”直接暴露出来。

## Decision

新增只读 endpoint：

- `GET /v1/agent-job-external-lease/cutover-diff`

它直接比较：

- `/v1/runtime-config`
- `/v1/queue-backend`
- `/v1/queue-topology`
- `/v1/agent-job-external-lease/launcher-bundle`

并输出：

- `ready/reason/blockers`
- `matching`
- `drift`
- `current_* / expected_*`
- embedded `bundle`

同时新增 repo-owned live verifier：

- `scripts/verify-agent-job-external-lease-cutover-diff.ps1`

它必须同时证明两段事实：

1. 当前 8780 live runtime 仍然 blocked，并能明确列出缺失的 flags、QueueDSN 和
   owner drift；
2. 用同一 canonical bundle 带起的 temp NATS/runtime，在造出 aged
   `group_memory_extract` pressure 后，会先表现为
   `worker_coverage_blocked + zero config drift`，再在 knowledge worker heartbeat
   后变成 `cutover_diff_ready + zero drift`。

## Requirements

- endpoint 必须保持 `side_effect=none`
- diff 必须显式覆盖：
  - `AKASHIC_QUEUE_BACKEND`
  - `AKASHIC_QUEUE_MODE`
  - `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER`
  - `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED`
  - `AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED`
  - `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED`
  - `AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED`
  - `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED`
  - `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN`
  - `QueueDSN`
  - `queue_provider`
  - `queue_mode`
  - `agent_job_execution_owner`
  - `agent_job_ack_owner`
  - `external_lease_ready`
- live verifier 必须证明当前 8780 runtime 的 blocked state 与 canonical bundle
  的 drift 可直接读出
- live verifier 必须证明 promoted temp runtime 在 worker coverage 之前已经消除
  config drift，但仍被 `agent_job_worker_coverage_blocked` 阻断
- promoted temp runtime 在 knowledge worker heartbeat 后必须变为
  `agent_job_external_lease_cutover_diff_ready`

## Non-Goals

- 不直接切当前 8780 runtime 的 `agent_job` production owner
- 不补 Telegram token
- 不解决 QQ native image blocker
- 不让 Go 执行 Python AI job

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_cutover_diff.py tests/test_verify_go_migration_goal_script.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-diff.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`
