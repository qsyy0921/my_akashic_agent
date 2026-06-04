# 227 Agent Job External Lease Launcher Bundle

## Context

`agent_job external lease result-ack` 当前已经有：

- `GET /v1/agent-job-external-lease/readiness`
- `GET /v1/agent-job-external-lease/plan`
- `GET /v1/agent-job-external-lease/preflight`
- repo-owned temp NATS smoke
- repo-owned isolated cutover preflight verifier
- repo-local launcher preflight smoke

但 launcher smoke 之前仍把启动参数硬编码在 verifier 里，没有一条 Go-owned
只读接口把 canonical launcher contract 直接暴露出来。

## Decision

新增只读 endpoint：

- `GET /v1/agent-job-external-lease/launcher-bundle`

它返回：

- `script_path`
- `launcher_parameters`
- `environment_overrides`
- `required_external_inputs`
- `verification_steps`
- 当前 `plan`
- `ready/reason/blockers`

并新增 repo-owned live smoke，直接读取这条 bundle，再用 bundle 带起 temp runtime，
验证 approval-bound preflight 与 ownership promotion。

## Requirements

- bundle 必须保持 `side_effect=none`。
- bundle 必须能覆盖 canonical result-ack launcher contract：
  - `QueueBackend=nats_jetstream`
  - `QueueMode=external_lease`
  - `QueueExternalLeaseCutover=true`
  - `QueueDualReadSmokePassed=true`
  - `QueueStateLeaseWorkersDisabled=true`
  - `QueueExternalLeaseAgentJobEnabled=true`
  - `QueueAgentJobDuplicateSmokePassed=true`
  - `QueueAgentJobFlowSmokePassed=true`
  - `AgentJobStrictLeaseToken=true`
- bundle 必须声明 operator-supplied external input：
  - `QueueDSN` / `AKASHIC_QUEUE_DSN`
- live smoke 必须真实证明：
  1. blocked runtime 上 bundle endpoint 可读，且 reason 为 blocked；
  2. bundle 暴露 canonical flags、script path 和 required external input；
  3. temp runtime 用 bundle 带起后，`runtime-config / queue-backend / queue-topology`
     体现 promoted state；
  4. worker coverage 满足后，bundle 自身变为 ready；
  5. approval-bound preflight 仍会从 `missing_approval_id` 走到
     `agent_job_external_lease_preflight_ready`。

## Non-Goals

- 不切当前 8780 production owner
- 不改 Telegram / QQ blocker
- 不新增 MQ adapter
- 不让 Go 执行 Python AI job

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_launcher_bundle.py tests/test_verify_go_migration_goal_script.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-launcher-bundle.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`
