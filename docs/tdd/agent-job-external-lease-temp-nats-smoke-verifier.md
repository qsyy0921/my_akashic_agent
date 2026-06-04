# Agent Job External Lease Temp NATS Smoke Verifier TDD

## Targeted Checks

- `services/agent-runtime/smoke/external_lease_nats_smoke_test.go`
  - smoke timestamps 不再依赖固定历史日期
- `scripts/verify_agent_job_external_lease_nats_smoke.py`
  - 覆盖 go-test JSON 解析
  - 覆盖 success / failure 结论归类
- `scripts/verify-go-migration-goal.ps1`
  - 覆盖 unified verifier 纳入新的 temp NATS smoke 证据

## Commands

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-nats-smoke.ps1`
- `uv run pytest tests/test_verify_agent_job_external_lease_nats_smoke.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
