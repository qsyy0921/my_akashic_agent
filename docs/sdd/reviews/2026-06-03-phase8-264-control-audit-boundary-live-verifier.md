# Review: Control Audit Boundary Live Verifier

## Scope

- repo-owned isolated live verifier for:
  - operator approvals
  - control mutations
  - approval check
  - control-mutation preflight
  - control-mutation policy
  - runtime-overview `control_mutation_policy` / `control_audit`

## What Changed

- Added `scripts/verify_control_audit_boundary.py`
- Added `scripts/verify-control-audit-boundary.ps1`
- Added `tests/test_verify_control_audit_boundary.py`
- Wired current-turn evidence into `scripts/verify-go-migration-goal.ps1`

## Verification

- `uv run pytest tests/test_verify_control_audit_boundary.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-control-audit-boundary.ps1`

## Outcome

- Control-audit ledger, preflight, policy, and runtime-overview parity now have
  repo-owned live evidence.
- Remaining gap is still executor absence, not approval/audit semantics.

## Residual Risk

- This verifier does not prove a real worker-control executor exists.
- Dashboard rendering for `Control Mutation Policy` / `Control Audit` remains a
  separate read-model concern, though current-turn panel evidence is now also
  captured by the unified goal verifier.
