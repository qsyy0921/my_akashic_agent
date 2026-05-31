# Review: agent_job external lease plan

Spec: `docs/sdd/specs/agent-gateway/076-agent-job-external-lease-plan.md`

Implementation summary:

- Added `PlanAgentJobExternalLeaseCommand`, `AgentJobExternalLeasePlanView` and plan steps.
- Added `AgentJobExternalLeasePlanService`, which reuses the existing readiness checker and produces read-only required checks, enable steps, verification steps and rollback steps.
- Added `GET /v1/agent-job-external-lease/plan` and wired it in `cmd/agent-runtime`.
- Documented that Go owns the deterministic result-ack plan while Python keeps AI execution.

Tests run:

- `go test ./app/service -run TestAgentJobExternalLeasePlan -count=1 -v`
- `go test ./trigger/http -run TestAgentJobExternalLease -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- No mutation path was added. The endpoint only returns plan data and never leases AgentJob records, acknowledges queue messages, starts workers or changes environment variables.
- Rollback is scoped to `agent_job` result-ack flags, so it does not accidentally disable the outbox external lease cutover path.

Decision:

- Accept. Operators now have a stable Go-owned plan endpoint before enabling generic AgentJob NATS result-ack.

Follow-ups:

- Run live comparison across `/v1/agent-job-external-lease/readiness`, `/v1/agent-job-external-lease/plan`, `/v1/queue-backend` and `/v1/runtime-overview` before any real result-ack cutover.
