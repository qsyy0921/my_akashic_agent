# SPEC-129: Runtime Overview Control Audit Table

## Status

Accepted for the current iteration.

## Context

Go owns operator approval and control mutation audit ledgers, and runtime
overview already aggregates `control_audit` as a card detail containing
`operator_approvals` and `control_mutations`. The dashboard plugin still renders
this as raw JSON, which makes it harder to inspect whether a future control-plane
mutation has a valid approval and what recent mutation attempts did.

This continues OI-008 by improving dashboard readability without adding control
actions.

## Boundary Analysis

Go owns:

- operator approval ledger;
- control mutation audit ledger;
- mutation policy/preflight and runtime overview control-audit data.

Dashboard owns:

- read-only presentation of the already loaded control-audit detail.

Out of scope:

- creating, revoking, or checking approvals;
- creating planned/applied/failed mutation records;
- executing cutover, worker scaling, queue ack/nack, media cleanup, or any AI
  work.

## Decision

When the runtime overview card id is `control_audit`, the dashboard panel
renders:

- approval totals and mutation totals;
- a bounded operator approvals table with approval id, target, decision, active
  state, operator and timestamp;
- a bounded control mutations table with mutation id, target/action, status,
  approval id, operator, reason/rollback and timestamp;
- raw JSON below as fallback/debug evidence.

The renderer consumes only `card.detail` and performs no network requests.

## Acceptance

- Plugin JS asset contains `Operator Approvals` and `Control Mutations` table
  sections.
- Existing runtime overview dashboard plugin tests remain green.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
