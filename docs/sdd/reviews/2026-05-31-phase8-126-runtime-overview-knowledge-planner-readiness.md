# Review: Runtime Overview Knowledge Planner Readiness

Spec:

- `docs/sdd/specs/agent-gateway/069-runtime-overview-knowledge-planner-readiness.md`

Implementation summary:

- Added optional `KnowledgeJobPlannerReadinessChecker` dependency to
  `RuntimeOverviewService`.
- Added `knowledge_job_planner_readiness` detail to runtime overview.
- Added summary fields for ready/blockers, planner enabled/running and Python
  knowledge worker active/stale/failed/stopped counts.
- Added `Knowledge Planner Readiness` card with muted / ok / warn states.
- Wired the existing readiness service into `cmd/agent-runtime`; no new worker
  or Python behavior was introduced.

Tests run:

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- Full package tests and build are required before commit.

Findings:

- Runtime overview now has both admission preview and admission readiness in one
  operator-facing aggregate.
- The readiness aggregation is read-only and uses the same env-derived planner
  defaults as the preview path.
- The Go/Python boundary remains unchanged: Go owns deterministic diagnostics;
  Python owns memory/RAG execution.

Decision:

- Accepted pending full regression. This is a safe observability-only extension
  to the existing readiness endpoint.

Follow-ups:

- Live smoke should compare `/v1/runtime-overview` readiness summary/card with
  `/v1/knowledge-job-planner/readiness` before enabling real planner admission.
