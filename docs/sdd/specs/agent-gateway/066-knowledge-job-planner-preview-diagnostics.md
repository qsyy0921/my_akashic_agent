# 066 Knowledge Job Planner Preview Diagnostics

## Context

`knowledge_job_planner` now lets Go own recurring observe-only QQ knowledge job
admission. That moves deterministic scheduling away from Python, but enabling a
new runtime planner still needs a safe preflight view.

Operators should be able to inspect which observe-only QQ groups and RAG
dataset bindings would produce `group_memory_extract` / `rag_ingest` jobs before
turning on real admission. The current endpoints show existing targets,
pipelines, jobs, and worker status, but they do not show the exact future
planner output without creating jobs.

## Decision

Add a read-only planner preview endpoint in Go:

```text
GET /v1/knowledge-job-planner/preview
```

The endpoint returns:

- planner timestamp bucket and interval;
- eligible observe-only QQ group targets;
- skipped targets with deterministic skip reasons;
- planned `group_memory_extract` and `rag_ingest` jobs;
- job id, job type, route, payload, dedupe key, max attempts, and metadata;
- aggregate totals for targets, skipped targets, group-memory jobs, RAG ingest
  jobs, and total jobs;
- `side_effect="none"`.

The preview reuses the same planning rules as the mutating
`PlanKnowledgeJobs` path. The mutating path should first build the same preview
plan, then persist each planned job through the existing AgentJob service.

## Boundary

### Go owns

- deterministic observe-target filtering;
- exact job id / dedupe key / route / payload planning;
- read-only planner diagnostics and HTTP API;
- default preview parameters derived from runtime env where practical.

### Python owns

- actual `group_memory_extract` execution;
- RAGFlow ingest execution;
- memory/RAG extraction algorithms, chunking, embedding, rerank and synthesis;
- provider-specific fallback and AI pipeline experimentation.

## Non-goals

- No change to Python worker execution.
- No automatic enabling of `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED`.
- No migration of RAGFlow API calls into Go.
- No queue-provider or scheduler rewrite.

## Validation

- Service tests prove preview plans the same target/job set without creating
  AgentJob records.
- HTTP tests prove the endpoint is GET-only, returns `side_effect="none"`, and
  exposes stable job plan fields.
- Targeted Go package tests and full `go test ./...` must pass.
