# Review: Knowledge Job Planner Readiness

Spec:

- `docs/sdd/specs/agent-gateway/068-knowledge-job-planner-readiness.md`

Implementation summary:

- Added `KnowledgeJobPlannerReadinessService` as a read-only app service.
- Added `GET /v1/knowledge-job-planner/readiness`.
- Readiness aggregates planner preview, runtime config, runtime worker
  diagnostics and Python `worker_type=knowledge` heartbeat state.
- Wired the service into `cmd/agent-runtime` using the same env-derived planner
  defaults as preview.
- Added service and HTTP tests for ready and blocked states.

Tests run:

- `go test ./app/service -run TestKnowledgeJobPlannerReadiness -count=1 -v`
- `go test ./trigger/http -run TestKnowledgeJobPlanner -count=1 -v`
- `go test ./cmd/agent-runtime -run TestKnowledgeJobPlanner -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:

- Disabled planner is intentionally not a blocker; readiness supports preflight
  before setting `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED=true`.
- Enabled planner with non-running runtime worker is blocked.
- No Python execution path or AgentJob state is mutated.

Decision:

- Accepted. The planner now has a clear Go-owned admission readiness gate.

Follow-ups:

- Optional future work can add operator ack/config mutation, but this slice
  stays read-only.
