# Review: Control Mutation Audit Ledger

Spec: `docs/sdd/specs/agent-gateway/093-control-mutation-audit-ledger.md`

Implementation summary:

- Added a Go-owned `ControlMutationAudit` domain model with planned/applied/failed/rolled_back statuses.
- Added app command/query/port/assembler/service layers for recording and listing control mutation audit records.
- Added file-backed infrastructure persistence at `.akashic-workspace/agent-runtime/control-mutations.json`, with `AKASHIC_CONTROL_MUTATIONS_DSN=memory` memory fallback.
- Added `POST /v1/control-mutations` and `GET /v1/control-mutations`.
- Wired the service into `cmd/agent-runtime/main.go`.
- Kept the endpoint as an audit ledger only: `side_effect=runtime_state_only`; no configuration mutation, no cutover, no AgentJob mutation, no MQ ack/nack, and no Python AI execution.

Tests run:

- `go test ./domain/model -run TestControlMutationAudit -count=1 -v`
- `go test ./app/service -run TestControlMutationAuditService -count=1 -v`
- `go test ./infrastructure/controlmutationstore -run TestStorePersistsControlMutationAudits -count=1 -v`
- `go test ./trigger/http -run TestControlMutationAuditEndpoints -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- Full regression commands are recorded in the iteration final response.

Findings:

- The ledger is intentionally not a mutation executor. It records planned and observed mutation state so future mutation endpoints can be audited without mixing execution logic into approval or plan endpoints.
- `failed` and `rolled_back` records require a reason. `rolled_back` also requires `rollback_of` or `rollback_ref`, so rollback history cannot be logged as an ambiguous status-only event.
- The API requires an `approval_id`, but it does not itself validate approval freshness. Future mutation endpoints should call approval check before recording applied/rolled_back outcomes.

Decision:

- Accept. This creates the missing audit surface needed before real control-plane mutations while preserving the current read-only safety boundary.

Follow-ups:

- Future mutation endpoints must require both approval preflight and mutation audit recording.
- Real authorization, replay protection and rollback execution semantics remain separate SDD tasks.
