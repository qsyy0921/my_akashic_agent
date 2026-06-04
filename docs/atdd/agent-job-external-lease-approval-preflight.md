# ATDD: Agent Job External Lease Approval Preflight

## Scenario 1: plan not ready

Given temp runtime 没有 external-lease / strict-token / result-ack flags

When 调用：

- `GET /v1/agent-job-external-lease/preflight?desired_execution_owner=python_ai_worker_with_nats_result_ack&operator_id=qsyy`

Then：

- `ready=false`
- `reason=agent_job_external_lease_plan_not_ready`
- `blockers` 含 plan blockers
- 当前 owner 仍是 `python_ai_worker_state_store_lease`

## Scenario 2: approval missing after ready plan

Given temp NATS runtime 已满足 readiness / plan / worker coverage

When 调用 preflight 但不传 `approval_id`

Then：

- `ready=false`
- `reason=missing_approval_id`
- `control_preflight.ready=false`

## Scenario 3: approval active after ready plan

Given temp NATS runtime 已满足 readiness / plan / worker coverage

And 已创建 active approval

When 调用带 `approval_id` 的 preflight

Then：

- `ready=true`
- `reason=agent_job_external_lease_preflight_ready`
- `suggested_audit` 可直接用于 planned mutation audit

## Guardrails

- 调用全程 `side_effect=none`
- 不改变 runtime flags
- 不切换长期运行 owner
- 不执行 Python AI
