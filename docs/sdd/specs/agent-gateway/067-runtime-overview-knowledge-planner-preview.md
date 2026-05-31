# 067 Runtime Overview Knowledge Planner Preview

## Context

`/v1/knowledge-job-planner/preview` can now show exactly which observe-only QQ
knowledge jobs would be admitted without mutating AgentJob state. The remaining
operational gap is discoverability: operators and the dashboard usually start
from `/v1/runtime-overview`, not from specialized feature endpoints.

Runtime overview already aggregates Go-owned queue, worker, observe, receiver,
knowledge pipeline and scheduler diagnostics. Planner preview should be visible
there as another read-only control-plane signal before enabling real
`knowledge_job_planner` admission.

## Decision

Add `KnowledgeJobPlannerPreview` as an optional dependency of
`RuntimeOverviewService`.

When present, runtime overview will:

- call `PreviewKnowledgeJobs` using env-derived planner defaults from `main`;
- include the full preview under `knowledge_job_planner_preview`;
- add summary fields for planned targets, skipped targets, groups, total jobs,
  `group_memory_extract` jobs and `rag_ingest` jobs;
- add a `knowledge_job_planner_preview` card with status `ok` when jobs are
  planned and `muted` when no eligible work exists.

The preview is optional and read-only. If the dependency is absent, runtime
overview remains compatible with existing test/service construction.

## Boundary

### Go owns

- overview aggregation of deterministic planner preview state;
- summary/card fields for dashboard and operator use;
- passing runtime env defaults to preview calculation.

### Python owns

- execution of `group_memory_extract` and `rag_ingest`;
- AI memory/RAG algorithms and provider-specific behavior;
- fallback enqueue behavior when Go planner is disabled or unavailable.

## Non-goals

- No Python changes.
- No dashboard UI changes in this slice.
- No enabling of real planner admission.
- No change to AgentJob lease, dedupe or retry semantics.

## Validation

- Runtime overview service test covers summary fields, card value/status and
  returned preview detail.
- `go test ./app/service`, `go test ./cmd/agent-runtime`, `go test ./...` and
  `go build ./cmd/agent-runtime` must pass.
