# Review: Phase 7.6 Go Agent Job Persistence

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/adr/0003-go-service-ddd-granularity.md`

Implementation summary:

- Added file-backed persistent `AgentJob` repository for gateway control-plane jobs:
  `services/agent-gateway/infrastructure/agentjobstore`.
- Added startup selector in `services/agent-gateway/cmd/agent-gateway/main.go`:
  - `AKASHIC_AGENT_JOBS_DSN` for explicit store path.
  - `AKASHIC_AGENT_JOBS_PATH` as shorthand path fallback.
  - default to in-memory when both env vars are empty.
- Wired `agentJobs` service to repository-selected store (not hardcoded in-memory store).
- Added repository unit tests validating persistence across restart and lifecycle state
  recovery.
- Updated gateway README with persistence env examples.
- Updated spec status in `008-agent-job-orchestration.md` for persistence completion.

Tests run:

- `go test ./...` (services/agent-gateway)
- targeted new test file under `services/agent-gateway/infrastructure/agentjobstore`

Findings:

- Persistence now makes job recovery possible after process restart.
- Concurrency is process-local; distributed multi-instance lease race prevention is
  still delegated to later infra work (single-writer or DB-backed CAS migration).

Decision:

- Acceptable as a persistence slice for current single-instance deployment.

Follow-ups:

- Add a dashboard view and/or SQL-backed repository if concurrent writers and richer
  querying are required.
