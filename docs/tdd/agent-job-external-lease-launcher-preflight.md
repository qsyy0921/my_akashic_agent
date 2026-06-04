# TDD: Agent Job External Lease Launcher Preflight

## Test additions

- `services/agent-runtime/cmd/agent-runtime/main_test.go`
  - runtime-config 环境变量可见性覆盖新增的 cutover/smoke booleans
- `tests/test_verify_agent_job_external_lease_launcher_preflight.py`
  - launcher 参数列表包含 external-lease / result-ack flags
  - verifier 成功时返回 `repo_owned_launcher_external_lease_preflight_live_verified`
- `tests/test_verify_go_migration_goal_script.py`
  - unified verifier 已接线 launcher preflight evidence

## Live smoke

- `scripts/verify-agent-job-external-lease-launcher-preflight.ps1`
  - 通过 `start-agent-runtime.ps1 -Foreground` 起隔离 temp runtime
  - 验证 runtime-config / queue-backend / queue-topology / approval-bound preflight
