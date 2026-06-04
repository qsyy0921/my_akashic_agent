# SDD: Control Audit Boundary Live Verifier

## Context

`operator approvals` and `control mutations` are already Go-owned control-plane
state, but before this slice the repo only had handler/service tests and
runtime-overview visibility. The remaining gap was repo-owned live evidence that
the approval ledger, mutation ledger, preflight, policy allowlist, and
runtime-overview summaries stay aligned in a real runtime without mutating
config or executing worker control.

## Decision

Add a repo-owned isolated verifier:

- `scripts/verify_control_audit_boundary.py`
- `scripts/verify-control-audit-boundary.ps1`

The verifier must:

1. start an isolated temp `agent-runtime`;
2. record approved/rejected operator approvals;
3. record planned/failed control-mutation audits;
4. verify approval checks, mutation preflight, and policy allowlist/blockers;
5. verify `.akashic-workspace/agent-runtime/operator-approvals.json` and
   `.akashic-workspace/agent-runtime/control-mutations.json` persist the same
   ledger state;
6. verify `/v1/runtime-overview` `control_mutation_policy` and `control_audit`
   summary/card/detail stay consistent with those ledgers.

## Invariants

- Approval and mutation endpoints may write runtime state only.
- Preflight and policy endpoints must keep `side_effect=none`.
- The verifier must not modify env/config, execute cutover, start worker control
  executors, ack/nack MQ, create/lease AgentJobs, or trigger Python AI work.
- A passing result proves control-audit boundary parity only. It does not imply
  a worker-control executor exists.

## Evidence

A successful verifier returns:

- `conclusion.status=live_verified`
- `conclusion.category=control_audit_boundary_live_verified`

and includes raw snapshots for:

- `policy`
- `operator_approvals`
- `control_mutations`
- `runtime_overview`

plus boolean parity checks for ledger persistence and overview alignment.
