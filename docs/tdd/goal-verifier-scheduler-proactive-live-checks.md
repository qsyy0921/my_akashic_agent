# Goal Verifier Scheduler Proactive Live Checks TDD

## Targeted Checks

- `scripts/verify-go-migration-goal.ps1`
  - 覆盖 scheduler jobs / leases / diagnostics 的 live 取证
  - 覆盖 proactive tick logs / quota / dashboard tick-log read model 的 live 取证
  - 覆盖 residual classification 不再输出 `not_current_turn`

## Regression Focus

- 不要破坏现有 QQ outbox scope、Telegram verifier、knowledge planner verifier
- 不要因为 scheduler 当前空闲就把它误写成“未验证”
- 不要把 proactive dashboard tick-log fallback 回退成纯文档性说明

## Commands

- `.\scripts\verify-go-migration-goal.ps1`
- `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
