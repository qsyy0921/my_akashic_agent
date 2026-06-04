# TDD: Dashboard Control Audit Boundary Live Verifier

## Scope

- Regression guardrail for the repo-owned dashboard control-audit boundary
  verifier and its unified-goal wiring.

## Target Code Paths

- `scripts/verify_dashboard_control_audit_boundary.py`
- `scripts/verify-dashboard-control-audit-boundary.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| dashboard matches runtime | unit | verifier checks all pass and result is `dashboard_control_audit_boundary_live_verified` |
| dashboard policy drifts | unit | verifier flags `dashboard_control_mutation_policy_matches_runtime=false` |
| runtime/dashboard fetch fails | unit | verifier result is `dashboard_control_audit_boundary_error` |

## Required Automated Tests

- `uv run pytest tests/test_verify_dashboard_control_audit_boundary.py -q`

## Deferred Coverage

- Live endpoint availability and real dashboard/runtime parity remain in the
  repo-owned live verifier path and unified goal verifier, not mocked unit
  tests.
