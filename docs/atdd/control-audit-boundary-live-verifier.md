# ATDD: Control Audit Boundary Live Verifier

## Scope

- Operator-visible control-audit state in Go runtime:
  - operator approvals ledger
  - control-mutation audit ledger
  - approval check
  - control-mutation preflight
  - control-mutation policy
  - runtime-overview control-audit visibility

## Preconditions

- Repo checkout available locally.
- `go` runnable for temp `agent-runtime`.
- `uv` or `python` available for the verifier script.

## Scenarios

### Scenario 1

- Action:
  Run `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-control-audit-boundary.ps1`.
- Expect:
  The verifier starts an isolated temp runtime, records approved/rejected
  approvals and planned/failed mutations, then returns
  `control_audit_boundary_live_verified`.

### Scenario 2

- Action:
  Inspect verifier evidence for approval check, mutation preflight, and policy.
- Expect:
  Active approval returns `approval_active`; missing/rejected cases return stable
  blockers; unsupported target/action remain blocked; policy stays read-only.

### Scenario 3

- Action:
  Inspect verifier evidence for runtime state files and runtime overview.
- Expect:
  `operator-approvals.json`, `control-mutations.json`, `/v1/runtime-overview`
  `control_mutation_policy`, and `/v1/runtime-overview` `control_audit`
  represent the same totals and recent records.

## Failure Signals

- Verifier returns `verification_failed` or `error`.
- Approval/mutation ledgers do not persist expected records.
- Runtime-overview control-audit summary/card/detail diverge from ledger totals.
- Any evidence shows config mutation, cutover execution, worker startup, MQ
  ack/nack, or Python AI side effects.

## Evidence

- Verifier JSON from `scripts/verify-control-audit-boundary.ps1`
- temp runtime logs and state-dir paths included in that JSON
