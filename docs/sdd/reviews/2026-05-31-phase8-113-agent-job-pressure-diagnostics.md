# Review: Agent Job Pressure Diagnostics

Spec:

- `docs/sdd/specs/agent-gateway/056-agent-job-pressure-diagnostics.md`

Implementation summary:

- Added Go-owned `pressure` diagnostics to `AgentJobMetricsView`.
- Aggregated job-type `pending`, `leased`, `running`, `active`, and
  `oldest_pending_age_seconds`, plus read-only `high_pressure` /
  `pressure_reason`.
- Added runtime overview summary fields:
  - `agent_job_pressure_job_types`
  - `agent_job_pressure_high_job_types`
  - `agent_job_pressure_max_pending`
  - `agent_job_pressure_max_active`
  - `agent_job_pressure_oldest_pending_age_seconds`
- Added `Agent Job Pressure` runtime overview card.
- Kept Python ownership unchanged: Python still executes knowledge, RAG, image,
  OCR/VLM, and other AI jobs; Go only reports deterministic backlog state.

Tests run:

- `go test ./app/service -run "TestAgentJobMetricsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./...`

Findings:

- No behavioral regression found in the implemented scope.
- Pressure diagnostics are purely read-only and do not mutate leasing, retry,
  or worker execution behavior.
- Knowledge/RAG job buildup is now visible without inferring it indirectly from
  dead letters or raw job lists.

Decision:

- Accept as Go-owned control-plane observability groundwork for future worker
  concurrency and scheduling policies.

Follow-ups:

- Live-check `/v1/job-metrics` and `/v1/runtime-overview` during real
  `group_memory_extract` / `rag_ingest` backlog.
- If future scheduling is added, keep enforcement/policy as a separate slice
  from these diagnostics.
