# Review: agent job external lease readiness

Spec: `docs/sdd/specs/agent-gateway/074-agent-job-external-lease-readiness.md`

Implementation summary:

- Added `GET /v1/agent-job-external-lease/readiness`.
- The endpoint aggregates queue backend external lease gate, runtime strict
  lease-token config, `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED`,
  AgentJob pressure and Python worker heartbeat coverage.
- Readiness is blocked when `agent_job` is not in allowed work kinds, result-ack
  owner is not enabled, strict tokens are disabled, the runtime flag is missing,
  or pressured job types have danger-level worker coverage.

Tests run:

- `go test ./app/service -run TestAgentJobExternalLeaseReadiness -count=1 -v`
- `go test ./trigger/http -run TestAgentJobExternalLeaseReadinessEndpointReturnsReadOnlyGate -count=1 -v`
- `go test ./cmd/agent-runtime -run "TestQueueBackend|TestRuntimeConfig" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `git diff --check` (only LF/CRLF warnings)

Findings:

- The endpoint is read-only. It does not lease AgentJobs, ack/nack NATS work,
  mutate env, start Python workers, or execute model/RAG/memory jobs.
- Python remains responsible for AI execution and reports worker status to Go.

Decision:

- Accepted as the next control-plane gate before any `agent_job` external lease
  result-ack cutover.

Follow-ups:

- Add runtime overview aggregation if the dashboard needs one-card visibility.
- Live result-ack cutover still requires NATS smoke and explicit operator flags.
