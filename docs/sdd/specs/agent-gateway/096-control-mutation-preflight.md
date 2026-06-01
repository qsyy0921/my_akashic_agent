# Control Mutation Preflight

## Context

Go already owns the operator approval ledger, approval check, control mutation
audit ledger and runtime overview control audit aggregate. The next safe step is
to make control-plane mutation execution paths call a deterministic preflight
before they mutate anything.

This slice does not implement real mutation execution. It only adds a read-only
preflight endpoint that checks whether a proposed mutation has an active
approved operator approval and returns the audit fields that the later mutation
executor must bind to.

## Ownership Boundary

Go owns:

- operator approval lookup and validation;
- mutation preflight decision;
- mutation audit binding shape;
- HTTP runtime API for control-plane preflight.

Python owns:

- AI worker execution;
- model/provider decisions;
- prompt/tool/RAG/image workflows;
- no control-plane approval or mutation authority.

Dashboard owns:

- optional read-only display of the preflight result;
- no preflight execution side effects.

## Goals

- Add a Go application use case for `control_mutation_preflight`.
- Validate required target/action/operator/approval fields.
- Reuse `OperatorApprovalManager.CheckOperatorApproval`.
- Return stable blockers when approval is missing, rejected, expired or target
  mismatched.
- Return a suggested `planned` mutation audit payload shape when ready.
- Expose `GET /v1/control-mutations/preflight`.

## Non-Goals

- No control-plane mutation execution.
- No mutation audit creation.
- No approval creation.
- No config/env/cutover changes.
- No worker startup, AgentJob mutation, MQ ack/nack, outbox delivery or AI
  execution.
- No dashboard redesign.

## API Sketch

```text
GET /v1/control-mutations/preflight
  ?target_kind=outbound_cutover
  &target_id=cutover-a
  &action=enable
  &operator_id=qsyy
  &approval_id=approval-a
```

Response data:

```json
{
  "ready": true,
  "reason": "approval_active",
  "blockers": [],
  "target_kind": "outbound_cutover",
  "target_id": "cutover-a",
  "action": "enable",
  "operator_id": "qsyy",
  "approval_id": "approval-a",
  "approval_check": {},
  "suggested_audit": {
    "target_kind": "outbound_cutover",
    "target_id": "cutover-a",
    "action": "enable",
    "status": "planned",
    "operator_id": "qsyy",
    "approval_id": "approval-a"
  },
  "side_effect": "none"
}
```

## Acceptance

- Service test covers ready path with an active approval.
- Service test covers blocked path when approval is missing.
- HTTP handler test covers `GET /v1/control-mutations/preflight`.
- Existing approval and mutation audit endpoints keep working.
- `go test ./app/service ./trigger/http` passes.
