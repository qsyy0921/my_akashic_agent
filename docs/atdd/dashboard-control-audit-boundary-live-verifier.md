# ATDD: Dashboard Control Audit Boundary Live Verifier

## Scope

- Operator-visible dashboard parity for:
  - `control_audit`
  - `control_mutation_policy`
  - top-level `operator_approvals`
  - top-level `control_mutations`

## Preconditions

- Local repo checkout available.
- Go runtime reachable at `http://127.0.0.1:8780`.
- Dashboard reachable at `http://127.0.0.1:2236`.
- `uv` or `python` available for the verifier script.

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-control-audit-boundary.ps1`.
- Expect:
  The verifier reads live runtime-overview and dashboard runtime-overview, then
  returns `dashboard_control_audit_boundary_live_verified`.

### Scenario 2

- Action:
  Inspect verifier evidence for `control_audit`.
- Expect:
  Dashboard summary totals, top-level `operator_approvals` /
  `control_mutations`, and `Control Audit` card all match the live Go runtime
  overview for the current turn.

### Scenario 3

- Action:
  Inspect verifier evidence for `control_mutation_policy`.
- Expect:
  Dashboard top-level policy detail and `Control Mutation Policy` card preserve
  the same allowlist, reason, totals, and side-effect contract as the live Go
  runtime overview.

## Failure Signals

- Verifier returns `verification_failed` or `error`.
- Dashboard drops the `control_audit` or `control_mutation_policy` card/detail.
- Dashboard summary totals drift from Go runtime-overview.
- Any evidence shows approval creation, mutation creation, config mutation,
  cutover execution, worker startup, MQ ack/nack, send side effects, or Python
  AI work.

## Evidence

- Verifier JSON from `scripts/verify-dashboard-control-audit-boundary.ps1`
- Current-turn unified goal verifier JSON showing
  `dashboard_read_models.control_audit_table=true` and
  `dashboard_read_models.control_mutation_policy_table=true`
