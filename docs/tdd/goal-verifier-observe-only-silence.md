# TDD: Goal Verifier Observe-Only Silence Check

## Coverage Targets

1. Unified goal verification includes current observe-only QQ group state.
2. Unified goal verification proves the runtime reply block with a live
   readiness call.
3. Unified goal verification still proves a private route is ready.

## Tests

- `.\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
