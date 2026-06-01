# Review: Operator Approval Ledger

Spec: `docs/sdd/specs/agent-gateway/091-operator-approval-ledger.md`

Implementation summary:

- Added a Go-owned `OperatorApproval` domain model for approved/rejected/revoked operator decisions.
- Added app command/query/port/assembler/service layers for recording and listing approvals.
- Added file-backed infrastructure persistence at `.akashic-workspace/agent-runtime/operator-approvals.json`, with `AKASHIC_OPERATOR_APPROVALS_DSN=memory` memory fallback.
- Added `POST /v1/operator-approvals` and `GET /v1/operator-approvals` HTTP routes in `services/agent-runtime`.
- Kept the API as an audit ledger only: `side_effect=runtime_state_only`; no configuration mutation, no cutover, no AgentJob mutation, no MQ ack/nack, and no Python AI execution.

Tests run:

- `go test ./domain/model -run TestOperatorApproval -count=1 -v`
- `go test ./app/service -run TestOperatorApprovalService -count=1 -v`
- `go test ./infrastructure/operatorapprovalstore -run TestStorePersistsOperatorApprovals -count=1 -v`
- `go test ./trigger/http -run TestOperatorApprovalEndpointsRecordAndListLedger -count=1 -v`
- Full regression commands are recorded in the iteration final response.

Findings:

- The ledger is intentionally append/overwrite-by-deterministic-id for a timestamped approval event; it is not current desired runtime state.
- Rejection and revocation require a reason, which prevents ambiguous negative approvals from entering the audit trail.
- Expired approvals remain queryable but are reported inactive.

Decision:

- Accept. This is the right Go-side prerequisite before future control-plane mutation work because it creates a durable, queryable acknowledgement boundary without allowing plan endpoints or dashboard clients to mutate runtime behavior.

Follow-ups:

- Future real cutover/capacity/priority mutations must require a valid approval id, emit configuration-change audit, and record rollback metadata.
- Do not let the ledger itself become an authorization system until permission, actor identity and replay rules are designed explicitly.
