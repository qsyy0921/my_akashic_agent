# TDD: Agent Job External Lease Launcher Bundle

## Test additions

- `services/agent-runtime/app/service/agent_job_external_lease_launcher_bundle_service_test.go`
  - canonical flags / required external inputs / blocked semantics
- `services/agent-runtime/trigger/http/handler_test.go`
  - `/v1/agent-job-external-lease/launcher-bundle` endpoint contract
- `tests/test_verify_agent_job_external_lease_launcher_bundle.py`
  - bundle-to-launcher argument building
  - verifier success category
- `tests/test_verify_go_migration_goal_script.py`
  - unified verifier wiring for launcher bundle evidence

## Live smoke

- `scripts/verify-agent-job-external-lease-launcher-bundle.ps1`
  - blocked temp runtime 读取 bundle
  - bundle 带起 promoted temp runtime
  - 验证 worker coverage / approval-bound preflight / runtime overview promoted state
