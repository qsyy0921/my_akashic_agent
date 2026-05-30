# Review: Phase 8.23 Agent Job Event Stream

Spec:
- `docs/sdd/specs/agent-gateway/012-agent-job-event-stream.md`

Implementation summary:
- Added `AgentJobEvent` domain model and lifecycle event types for generic job
  transitions.
- Added app query/assembler/inbound/outbound ports for job event listing.
- Extended `AgentJobService` with an optional event sink and wired the runtime
  constructor to append events on create, lease, running, succeeded, failed,
  retry, and cancelled transitions.
- Added `infrastructure/agentjobeventstore` JSONL adapter as the first durable
  stream backend.
- Added `/v1/job-events` as a read-only event stream query API.
- Wired runtime configuration through `AKASHIC_AGENT_JOB_EVENTS_DSN` and
  `AKASHIC_AGENT_JOB_EVENTS_PATH`.

Tests run:
- `go test ./...` under `services/agent-runtime`
- `go build -o ..\..\.tmp\bin\agent-runtime-new.exe .\cmd\agent-runtime`
- Live smoke against `127.0.0.1:8780`: create, lease, and cancel a `rag_eval`
  job, then verify `/v1/job-events?job_id=...` returns `created`, `leased`, and
  `cancelled` events.
- `git diff --check -- services\agent-runtime docs\sdd`

Decision:
- Approved as the first local durable stream slice. This does not replace the
  job repository or Python worker contracts; it creates a stable event boundary
  for dashboard diagnostics and future NATS/Redis Streams adapters.

Follow-ups:
- Add dashboard visibility for `/v1/job-events`.
- Decide whether outbox delivery state should use the same event stream pattern
  before introducing an external queue backend.
