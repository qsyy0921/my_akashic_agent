# 226 Agent Job External Lease Launcher Preflight

## Context

`agent_job external lease result-ack` 已有：

- `GET /v1/agent-job-external-lease/readiness`
- `GET /v1/agent-job-external-lease/plan`
- `GET /v1/agent-job-external-lease/preflight`
- repo-owned temp NATS smoke
- repo-owned isolated cutover preflight verifier

但 repo-local launcher `scripts/start-agent-runtime.ps1` 还不能显式注入
external-lease / result-ack flags，也没有一条通过 launcher 自身起 runtime 的
repo-owned live smoke。

## Decision

补齐两层能力：

1. `scripts/start-agent-runtime.ps1` 新增 external-lease / result-ack 相关参数：
   - runtime addr / state dir / shadow path
   - queue backend / dsn / mode / concurrency / max-in-flight
   - external-lease cutover / dual-read / state-lease-workers-disabled
   - agent-job external-lease enable / duplicate smoke / flow smoke
   - strict lease token
2. 新增 repo-owned launcher live smoke：
   - `scripts/verify-agent-job-external-lease-launcher-preflight.ps1`
   - 使用 `start-agent-runtime.ps1 -Foreground` 在隔离 temp runtime 上验证
     launcher 参数注入，而不是依赖手工 env export

## Requirements

- launcher 必须支持显式覆盖：
  - `AKASHIC_RUNTIME_ADDR`
  - `AKASHIC_RUNTIME_STATE_DIR`
  - `AKASHIC_SHADOW_AUDIT_PATH`
  - `AKASHIC_QUEUE_BACKEND`
  - `AKASHIC_QUEUE_DSN`
  - `AKASHIC_QUEUE_MODE`
  - `AKASHIC_QUEUE_CONSUMER_CONCURRENCY`
  - `AKASHIC_QUEUE_MAX_IN_FLIGHT`
  - `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER`
  - `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED`
  - `AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED`
  - `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED`
  - `AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED`
  - `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED`
  - `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN`
- `/v1/runtime-config` 必须可见上述 cutover/smoke booleans，且按非 secret 布尔配置脱敏展示。
- launcher live smoke 必须真实证明：
  - runtime-config 里这些 flags 已 present
  - queue backend 已提升到 `nats_jetstream`
  - queue topology 的 `agent_job` execution owner / ack owner 已提升
  - preflight 无 approval 时为 `missing_approval_id`
  - active approval 后为 `agent_job_external_lease_preflight_ready`

## Non-Goals

- 不执行 production cutover
- 不改当前 8780 live runtime flags
- 不让 Go 执行 Python AI job
- 不新增 MQ adapter

## Verification

- `go test ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_launcher_preflight.py tests/test_verify_go_migration_goal_script.py -q`
- `uv run python -c "...run_agent_job_external_lease_launcher_preflight(...)"` 小输出直调
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-launcher-preflight.ps1 | Set-Content ...`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`
