# Review: Dashboard Control Audit Boundary Live Verifier

## Scope

- repo-owned live verifier for dashboard parity of:
  - `control_audit`
  - `control_mutation_policy`
  - top-level `operator_approvals`
  - top-level `control_mutations`

## What Changed

- Added `scripts/verify_dashboard_control_audit_boundary.py`
- Added `scripts/verify-dashboard-control-audit-boundary.ps1`
- Added `tests/test_verify_dashboard_control_audit_boundary.py`
- Wired current-turn evidence into `scripts/verify-go-migration-goal.ps1`
- Replaced unified-goal `control_audit` / `control_mutation_policy` dashboard
  proof with live dashboard/runtime parity instead of static panel substring
  probes

## Verification

- `uv run pytest tests/test_verify_dashboard_control_audit_boundary.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-control-audit-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Outcome

- Dashboard `control_audit` and `control_mutation_policy` now have repo-owned
  live parity evidence against the Go runtime overview.
- Unified goal verification no longer needs to treat these two read-models as
  panel-string-only evidence.

## Residual Risk

- This verifier does not prove a real worker-control executor exists.
- Future dashboard gaps can still appear on newly added read-models outside this
  control-audit boundary.
