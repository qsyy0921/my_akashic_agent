# 069 Runtime Overview Knowledge Planner Readiness

## Context

`/v1/knowledge-job-planner/readiness` already provides a read-only gate for
turning on Go-owned observe-only knowledge job admission. Runtime overview
already exposes the planner preview, but operators still need to open a second
endpoint to see whether the planner is safe to enable.

This slice brings the readiness result into `/v1/runtime-overview` so the
dashboard and operator entrypoint can answer both questions together:

- what knowledge jobs would be planned;
- whether the planner can be enabled without immediately losing execution
  capacity or silently doing nothing.

## Decision

Add optional runtime overview aggregation for
`KnowledgeJobPlannerReadinessChecker`.

Runtime overview will expose:

- `knowledge_job_planner_readiness` detail object;
- summary fields for ready/blocker count, planner enabled/running, worker
  readiness, worker active/stale/failed/stopped counts;
- a `Knowledge Planner Readiness` card with `ok` when ready, `warn` when
  blocked, and `muted` when no readiness checker is configured.

The aggregation remains read-only. It reuses the same default planner command
used by preview and does not create AgentJob records, start workers, mutate env
vars, or change Python worker behavior.

## Boundary

### Go owns

- readiness aggregation into runtime overview;
- stable dashboard-facing summary/card fields;
- optional dependency handling and error reporting;
- command wiring from `cmd/agent-runtime`.

### Python owns

- actual `group_memory_extract` / `rag_ingest` execution;
- worker heartbeat lifecycle reporting;
- model, prompt, memory, RAG, OCR/VLM and provider-specific behavior.

## Non-goals

- No automatic enabling of `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED`.
- No new worker scheduling or queue mutation.
- No Python code changes.
- No frontend layout changes in this slice.

## Validation

- Runtime overview service test covers readiness detail, summary fields and
  card state.
- Existing HTTP/main wiring tests must still compile.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check` must
  pass.
