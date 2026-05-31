# Review: Queue Execution Owner Diagnostics

Spec:

- `docs/sdd/specs/agent-gateway/055-queue-execution-owner-diagnostics.md`

Implementation summary:

- Added explicit execution owner fields to Go `/v1/queue-backend`:
  - `outbox_execution_owner`
  - `agent_job_execution_owner`
- Added matching runtime overview summary fields:
  - `queue_outbox_execution_owner`
  - `queue_agent_job_execution_owner`
- Kept behavior read-only: no queue cutover, no worker ownership change, no Python AI execution migration.
- Clarified Python AI runtime responsibilities in the spec and iteration prompt: Python owns model/provider routing, prompt/context pipeline, tool orchestration, Memory/RAG algorithms, OCR/VLM, image generation, group knowledge extraction, evaluation and provider fallback; Go owns deterministic lifecycle, queue, lease, idempotency, audit and diagnostics.
- Tightened iteration governance: current `TODO.md` must be fully completed before closing an iteration; newly added in-iteration requirements are written into TODO and completed in the same loop.

Tests run:

- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*(DefaultsLocal|ExternalLease|OutboxWorker|AgentJobResultAck)" -count=1 -v`
- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./...`

Findings:

- No behavioral regression found.
- Owner diagnostics correctly distinguish Go local outbox worker, Go state-store API exposure, NATS external lease outbox execution and Python AI worker agent_job execution.
- NATS agent_job result-ack does not imply Go executes AI work; the exposed owner string makes that boundary explicit.

Decision:

- Accept this slice as a read-only diagnostics and SDD governance improvement.

Follow-ups:

- Live-check `/v1/queue-backend` and `/v1/runtime-overview.summary` under default, local worker, external lease and agent_job result-ack configurations.
- If dashboard UX needs it, build a dedicated queue topology panel later rather than expanding this iteration.
