# TDD: Proactive Runtime Flow Live Smoke

## Scope

- 保护 repo-owned proactive live verifier 的 state-file summarization、
  conclusion 分类和 unified goal wiring。

## Target Code Paths

- `scripts/verify_proactive_runtime_flow_live_smoke.py`
- `scripts/verify-proactive-runtime-flow-live-smoke.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| missing state file | unit | state counts default to zero |
| populated state file | unit | state counts map all proactive keys correctly |
| all smoke checks pass | unit | result category is `go_proactive_state_flow_live_verified` |
| one smoke check fails | unit | result category is `proactive_state_flow_verification_incomplete` |
| unified verifier wiring | contract | goal verifier references proactive flow smoke wrapper and result fields |

## Required Automated Tests

- `uv run pytest tests/test_verify_proactive_runtime_flow_live_smoke.py tests/test_verify_go_migration_goal_script.py -q`

## Deferred Coverage

- temp runtime 的完整 live path 由
  `scripts/verify-proactive-runtime-flow-live-smoke.ps1`
  负责，不放进常规 pytest。
