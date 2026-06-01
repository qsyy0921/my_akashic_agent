# Runtime Overview Control Audit

## Context

Go now owns three control-plane audit surfaces:

- operator approval ledger;
- operator approval check preflight;
- control mutation audit ledger.

Operators should not need to manually call separate endpoints to understand
whether recent control-plane approvals and mutation audits exist. This slice
adds a read-only runtime overview aggregate for these audit surfaces.

## Ownership Boundary

Go owns:

- deterministic aggregation of approval and mutation audit records;
- runtime overview summary/card/detail fields;
- error isolation if an audit reader is unavailable.

Python owns:

- dashboard display of the Go aggregate;
- AI execution, model calls, prompt/tool orchestration and RAG/Memory pipelines.

## Goals

- Add control audit dependencies to `RuntimeOverviewService`.
- Aggregate operator approvals and control mutation audits into summary/card/detail.
- Expose counts for total approvals, active approvals, approved/rejected/revoked,
  total mutations, planned/applied/failed/rolled_back, and bounded recent
  records.
- Keep the aggregate read-only.

## Non-Goals

- No approval creation.
- No mutation audit creation.
- No approval preflight execution.
- No configuration mutation.
- No cutover execution.
- No worker startup, queue ack/nack/term, AgentJob mutation or Python AI
  execution.

## Summary Fields

- `operator_approvals_total`
- `operator_approvals_active`
- `operator_approvals_approved`
- `operator_approvals_rejected`
- `operator_approvals_revoked`
- `control_mutations_total`
- `control_mutations_planned`
- `control_mutations_applied`
- `control_mutations_failed`
- `control_mutations_rolled_back`

## Cards

`Control Audit`

- `ok`: approvals exist and no failed/rolled_back mutation audits are present.
- `warn`: no approvals exist, or failed/rolled_back mutation audits are present.
- `unknown`: both audit readers are unavailable.

## Acceptance

- Runtime overview service test covers summary/card/detail.
- Full Go tests remain green.
- SDD DONE/LIVE/BACKLOG/review/TODO are updated.
