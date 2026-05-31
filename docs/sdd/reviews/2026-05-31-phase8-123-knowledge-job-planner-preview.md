# Review: Knowledge Job Planner Preview

Spec:

- `docs/sdd/specs/agent-gateway/066-knowledge-job-planner-preview-diagnostics.md`

Implementation summary:

- Added Go `PreviewKnowledgeJobs` on `KnowledgeJobPlannerService`.
- Refactored mutating `PlanKnowledgeJobs` to build the same preview plan before
  creating AgentJob records, reducing drift between diagnostics and admission.
- Added `GET /v1/knowledge-job-planner/preview` with env-derived defaults and
  query overrides for timestamp, interval, planner id, agent id, attempts,
  RAG max messages, and RAG parse flag.
- Added preview query DTOs and an app-layer input port.
- Updated runtime README and SDD tracking docs.

Tests run:

- `go test ./app/service -run TestKnowledgeJobPlanner -count=1 -v`
- `go test ./trigger/http -run TestKnowledgeJobPlannerPreviewEndpoint -count=1 -v`
- `go test ./cmd/agent-runtime -run TestKnowledgeJobPlanner -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- Preview remains strictly read-only and does not require Python changes.
- Go owns deterministic planning and skip reasons; Python remains the execution
  worker for memory/RAG AI pipeline jobs.

Decision:

- Accepted. The endpoint is a safe preflight gate before enabling real
  `knowledge_job_planner` admission.

Follow-ups:

- Optional dashboard panel can consume this endpoint later, but it is not part
  of this round.
