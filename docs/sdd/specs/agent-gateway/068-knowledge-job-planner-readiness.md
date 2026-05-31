# 068 Knowledge Job Planner Readiness

## Context

Go now owns recurring observe-only knowledge job admission through
`knowledge_job_planner`, and it exposes a read-only preview. Runtime overview
also surfaces that preview. The next operational gap is the actual admission
gate: before turning the planner on, operators need one endpoint that explains
whether admission is safe and what is blocking it.

Preview alone answers "what would be planned". Readiness should answer "can we
admit this work without immediately losing execution capacity or silently doing
nothing".

## Decision

Add a read-only readiness endpoint:

```text
GET /v1/knowledge-job-planner/readiness
```

The endpoint will aggregate:

- `PreviewKnowledgeJobs` output;
- runtime config `knowledge_job_planner_enabled`;
- runtime worker diagnostics for `knowledge_job_planner`;
- Python AI worker status for `worker_type=knowledge`.

It returns:

- `ready`, `reason`, `blockers`;
- planned targets / groups / total jobs inherited from preview;
- whether the planner is currently enabled and running;
- counts for active, stale, failed and stopped knowledge workers;
- `side_effect="none"`.

Readiness does not require the planner to already be enabled. It treats a
disabled planner as a preflight state, not a blocker. If the planner is enabled
but the Go runtime worker is not running, readiness is blocked. If no jobs would
be planned or no active Python knowledge worker is available, readiness is also
blocked.

## Boundary

### Go owns

- planner admission preflight;
- deterministic blockers and readiness reason;
- worker status aggregation and runtime worker state checks;
- HTTP API for operator/dashboard consumption.

### Python owns

- actual `group_memory_extract` and `rag_ingest` execution;
- worker heartbeat reporting;
- memory/RAG algorithms, provider fallback and strategy experiments.

## Non-goals

- No automatic enabling of `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED`.
- No mutation of AgentJob state.
- No Python code changes.
- No dashboard UI changes in this slice.

## Validation

- Service tests cover ready and blocked states.
- HTTP tests prove the endpoint is GET-only, returns readiness blockers, and is
  read-only.
- `go test ./app/service`, `go test ./trigger/http`, `go test ./...`, and
  `go build ./cmd/agent-runtime` must pass.
