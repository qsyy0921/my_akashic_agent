# TDD: Control Audit Boundary Live Verifier

## Scope

- Regression guardrail for the repo-owned control-audit boundary verifier result
  classification.

## Target Code Paths

- `scripts/verify_control_audit_boundary.py`
- `scripts/verify-control-audit-boundary.ps1`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| all checks pass | unit | verifier result is `control_audit_boundary_live_verified` |
| one parity check fails | unit | verifier result is `control_audit_boundary_verification_failed` |
| runtime setup errors | unit | verifier result is `control_audit_boundary_error` |

## Required Automated Tests

- `uv run pytest tests/test_verify_control_audit_boundary.py -q`

## Deferred Coverage

- Temp runtime bring-up, HTTP behavior, and state-file persistence stay in the
  repo-owned live smoke path and are verified by
  `scripts/verify-control-audit-boundary.ps1`, not mocked unit tests.
