# SDD: Dashboard Control Audit Boundary Live Verifier

## Context

Go already owns `operator_approvals`, `control_mutations`, and
`control_mutation_policy`, and the previous slice added repo-owned live evidence
for the runtime endpoints and runtime-overview parity. The remaining weak point
was dashboard evidence: unified goal verification still relied on static
`dashboard_panel.js` substring probes for `control_audit` and
`control_mutation_policy`, which did not prove the live dashboard read-model was
still aligned with Go runtime output.

## Decision

Add a repo-owned live verifier:

- `scripts/verify_dashboard_control_audit_boundary.py`
- `scripts/verify-dashboard-control-audit-boundary.ps1`

The verifier must:

1. read live Go `/v1/runtime-overview`;
2. read live dashboard `/api/dashboard/runtime-overview`;
3. compare control-audit summary fields;
4. compare top-level `operator_approvals` and `control_mutations` detail;
5. compare `control_audit` card value/status/detail;
6. compare `control_mutation_policy` top-level detail and card value/status.

## Invariants

- The verifier is read-only and must not create approvals or mutations.
- The verifier must not modify env/config, execute cutover, start worker
  executors, ack/nack MQ, create/lease AgentJobs, send platform messages, or
  trigger Python AI work.
- A passing result proves live dashboard parity for this read-model boundary
  only. It does not imply a real worker-control executor exists.

## Evidence

A successful verifier returns:

- `conclusion.status=live_verified`
- `conclusion.category=dashboard_control_audit_boundary_live_verified`

and includes normalized snapshots for:

- runtime-overview summary/card/detail
- dashboard runtime-overview summary/card/detail
- boolean parity checks for control-audit and control-mutation-policy fields

Unified goal verification must consume this verifier output instead of relying
only on static panel-string probes for these two dashboard read-models.
