# 225 Agent Job External Lease Approval Preflight

## Context

Go runtime 已有：

- `/v1/agent-job-external-lease/readiness`
- `/v1/agent-job-external-lease/plan`
- repo-owned temp NATS smoke
- repo-owned isolated cutover preflight verifier

但还缺一个稳定的 approval-bound API，把 `agent_job external lease result-ack`
的 cutover gate 收敛成可直接调用的只读 preflight，而不是让调用方自己拼
`plan + control-mutations/preflight`。

## Decision

新增只读 endpoint：

- `GET /v1/agent-job-external-lease/preflight`

语义：

- `target_kind=agent_job_external_lease`
- `action=enable`
- 默认 `desired_execution_owner=python_ai_worker_with_nats_result_ack`
- 默认 `target_id=<desired_execution_owner>`
- 先读取 `plan`
- 只有在 `plan.ready=true` 后，才继续做 control-mutation approval preflight

## Requirements

- 当 `plan.ready=false` 时：
  - 返回 `ready=false`
  - `reason=agent_job_external_lease_plan_not_ready`
  - `blockers` 直接反映 plan blockers
  - 不调用 control mutation approval preflight
- 当 `plan.ready=true` 且没有 approval 时：
  - 返回 `ready=false`
  - `reason=missing_approval_id`
  - detail 必须带 `control_preflight`
- 当 `plan.ready=true` 且 approval active 时：
  - 返回 `ready=true`
  - `reason=agent_job_external_lease_preflight_ready`
  - detail 必须带 `suggested_audit`
- endpoint 保持 `side_effect=none`
- 不修改 queue backend、flags、owner、worker、approval ledger、control mutation ledger

## Non-Goals

- 不直接执行 production cutover
- 不创建 approval
- 不写 control mutation audit
- 不启动 Python worker
- 不 ack/nack MQ
- 不触发任何 Python AI side effect

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_cutover_preflight.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-preflight.ps1`
