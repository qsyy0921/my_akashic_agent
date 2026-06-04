# Agent Job External Lease Live Verifier TDD

## Targeted Checks

- `scripts/verify-agent-job-external-lease.ps1`
  - 覆盖 readiness / plan / runtime-config / queue-backend / queue-topology /
    runtime-overview / agent-worker-statuses 的 live 聚合
  - 覆盖 blocker bucket 分类
  - 覆盖 `current_scope` 与 `conclusion.category`
- `scripts/verify-go-migration-goal.ps1`
  - 覆盖 unified verifier 复用新的 agent-job verifier

## Commands

- `.\scripts\verify-agent-job-external-lease.ps1`
- `.\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
