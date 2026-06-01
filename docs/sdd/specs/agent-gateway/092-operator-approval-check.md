# Operator Approval Check

## Context

The Go-owned operator approval ledger records human acknowledgement, but future
runtime control-plane mutations also need a stable way to validate whether a
specific approval can be used for a specific target before any mutation runs.

This slice adds a read-only approval check endpoint. It does not perform the
mutation itself. It is the small control-plane contract that future cutover,
capacity, priority or worker-concurrency mutation code can depend on.

## Ownership Boundary

Go owns:

- approval lookup and validation against the Go-owned ledger;
- active/expired decision evaluation;
- HTTP API for deterministic preflight checks;
- side-effect declaration and machine-readable blockers.

Python owns:

- AI execution, model calls, prompt/tool orchestration;
- dashboard display clients that may call this check;
- future AI worker behavior after a Go-approved mutation is separately
  designed.

## Goals

- Add a read-only approval check use case to the existing operator approval
  service.
- Add `GET /v1/operator-approvals/check`.
- Support checks by either:
  - `approval_id`; or
  - `target_kind` + `target_id`, using the newest matching ledger record.
- Return machine-readable `approved`, `reason`, `blockers`, `approval` and
  `side_effect=none`.

## Non-Goals

- No configuration mutation.
- No cutover execution.
- No autoscaling, priority mutation or worker startup.
- No queue ack/nack/term.
- No AgentJob mutation.
- No Python AI execution.
- No authorization/identity system.

## API

`GET /v1/operator-approvals/check`

Query fields:

- `approval_id`: optional exact approval record id.
- `target_kind`: required when `approval_id` is omitted.
- `target_id`: required when `approval_id` is omitted.

Response data:

- `approved`: boolean.
- `reason`: stable reason string.
- `blockers`: list of stable blocker strings.
- `approval`: matched approval view when found.
- `side_effect`: always `none`.
- `notes`: human-readable notes.

## Decision Rules

- Missing lookup fields returns `approved=false` with
  `missing_approval_lookup`.
- No matching ledger record returns `approval_not_found`.
- If `approval_id` is provided with a mismatched `target_kind` or `target_id`,
  return `approval_target_mismatch`.
- `decision != approved` returns `approval_not_approved`.
- Expired approval returns `approval_expired`.
- An active approved record returns `approved=true`.

## Acceptance

- Service tests cover active approved, missing, rejected and expired approvals.
- HTTP tests cover successful and blocked checks.
- Full Go tests remain green.
