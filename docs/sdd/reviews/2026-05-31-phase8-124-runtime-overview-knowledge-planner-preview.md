# Review: Runtime Overview Knowledge Planner Preview

Spec:

- `docs/sdd/specs/agent-gateway/067-runtime-overview-knowledge-planner-preview.md`

Implementation summary:

- Added optional `KnowledgeJobPlanner` preview dependency to
  `RuntimeOverviewService`.
- Wired `cmd/agent-runtime` so runtime overview uses the same env-derived
  planner defaults as `/v1/knowledge-job-planner/preview`.
- Added runtime overview summary fields and a `Knowledge Planner` card for
  planned targets, skipped targets, groups, total jobs, group-memory jobs and
  RAG ingest jobs.
- Exposed the full preview detail as `knowledge_job_planner_preview` in the
  overview response.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./cmd/agent-runtime -run TestKnowledgeJobPlannerPreviewCommandFromEnv -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- The aggregation is read-only and optional. Missing preview dependency does not
  make runtime overview partial.
- Python remains responsible for execution of memory/RAG jobs and AI pipeline
  behavior.

Decision:

- Accepted. Runtime overview can now act as the operator entrypoint for
  knowledge planner preflight without requiring dashboard code changes.

Follow-ups:

- A dashboard drilldown can consume the overview detail later if needed.
