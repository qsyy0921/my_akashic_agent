# Review: Phase 7.1 Go Agent Job Control Plane

Spec:

- `docs/sdd/specs/message-gateway/008-agent-job-orchestration.md`
- `docs/sdd/adr/0003-go-service-ddd-granularity.md`

Implementation summary:

- Added Go `AgentJob` domain lifecycle for pending, leased, running, succeeded,
  failed, dead-lettered, and cancelled states.
- Added lease expiry semantics and bounded attempts.
- Added app-layer command/query/ports/service for generic jobs.
- Added in-memory repository/query support in the gateway store.
- Added `/v1/jobs` and `/v1/jobs/lease-next` HTTP control-plane endpoints.
- Kept Python as the executor for image generation, RAG, and group-memory
  extraction.

Tests run:

- `gofmt -w api app cmd domain infrastructure trigger types`
- `go test ./...`

Findings:

- The job store is still in-memory. This is enough for domain/API validation but
  not durable production work.
- Existing image-specific job APIs still exist. They should become
  compatibility wrappers or be backed by generic `AgentJob` later.
- Python worker client integration is not wired yet.

Decision:

- Accept as a Phase 7.1 control-plane slice.

Follow-ups:

- Add persistence for generic jobs.
- Add Python worker compatibility client tests.
- Route image generation jobs through generic `AgentJob` while preserving
  current user behavior.
- Route group-memory and RAG ingestion through generic jobs in observe-only
  mode.
