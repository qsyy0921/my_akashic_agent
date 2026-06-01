# Dashboard Control Audit Detail

## Context

Go runtime overview now exposes `control_audit` card/detail plus raw
`operator_approvals` and `control_mutations` aggregate data. The Python
dashboard runtime overview plugin currently normalizes known Go-owned details
for a stable frontend API. It should do the same for control audit so the
browser/UI can consume predictable fields without owning any control-plane
state.

## Ownership Boundary

Go owns:

- operator approval ledger;
- approval check;
- control mutation audit ledger;
- runtime overview aggregation.

Python owns:

- dashboard read-only display normalization;
- HTTP proxying to Go runtime overview;
- no control-plane mutation state.

## Goals

- Normalize `operator_approvals` in `/api/dashboard/runtime-overview`.
- Normalize `control_mutations` in `/api/dashboard/runtime-overview`.
- Add summary defaults for control audit counters.
- Preserve the `control_audit` card/detail from Go.

## Non-Goals

- No approval creation.
- No mutation audit creation.
- No approval check invocation.
- No config mutation, cutover, worker startup, AgentJob mutation, MQ ack/nack or
  Python AI execution.
- No dashboard UI redesign.

## Acceptance

- Dashboard plugin test verifies:
  - `control_audit` card is present;
  - summary defaults expose operator approval and control mutation counters;
  - normalized `operator_approvals` and `control_mutations` details contain
    stable totals and recent records;
  - `side_effect` remains read-only.
- Targeted Python test passes.
