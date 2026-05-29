# Review: Agent Job Orchestration SDD

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/adr/0003-go-service-ddd-granularity.md`

Findings:

### P0

None.

### P1

1. Job orchestration must not move AI execution to Go. The spec keeps image
   generation, vision, embeddings, RAG ranking, and memory extraction in Python
   workers, which is correct.
2. Observe-only group jobs must be enforced at job creation time, not only in
   Python worker prompts.

### P2

1. The first implementation should avoid NATS/Redis until the domain/app API is
   proven with in-memory tests.
2. Existing image-job endpoints can either remain as compatibility wrappers or
   be backed by generic `AgentJob`; the migration should avoid breaking current
   image generation calls.

Decision:

**Design passes for a Go job lifecycle control-plane slice.** Implement generic
job lifecycle next, but keep current image/RAG/memory Python execution.

Follow-ups:

- Add `AgentJob` domain lifecycle and lease semantics.
- Add HTTP endpoints for create/list/get/lease/update.
- Add Python worker client tests before routing real image/RAG work through the
  new job API.
