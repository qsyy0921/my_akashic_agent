# Review 282

## Scope

- Go `agent_job external lease` approval-bound preflight endpoint
- service / handler / verifier / tests

## What changed

- 新增 `GET /v1/agent-job-external-lease/preflight`
- preflight 先吃 `plan`，再做 approval-bound control preflight
- isolated cutover preflight verifier 现在额外固定：
  - plan 未 ready 时直接返回 `agent_job_external_lease_plan_not_ready`
  - plan ready 但没 approval 时返回 `missing_approval_id`
  - approval active 后返回 `agent_job_external_lease_preflight_ready`

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_cutover_preflight.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-preflight.ps1`

## Residual

- production runtime 仍未切到 external lease result-ack
- blocker 仍是 flags / owner / NATS production cutover，而不是 preflight 缺失
