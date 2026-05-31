# Review: Go-Owned Knowledge Job Planner

Spec:

- `docs/sdd/specs/agent-gateway/065-go-owned-knowledge-job-planner.md`

Implementation summary:

- Added Go `KnowledgeJobPlannerService` under `app/service`.
- Added a `trigger/job` runtime worker that periodically calls the planner.
- Added `cmd/agent-runtime` env wiring:
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_PARSE`
  - `AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START`
- Runtime config and runtime worker diagnostics now expose the planner flag and worker.
- Python `AgentGatewayKnowledgeWorker` now probes `/v1/runtime-config`; when Go reports `knowledge_job_planner_enabled=true`, Python skips legacy enqueue and only leases/executes knowledge jobs.
- Fixed `bootstrap.app` knowledge worker construction so the worker is created even when RAGFlow is disabled.

Boundary:

- Go owns deterministic observe-only knowledge job admission, interval, dedupe-aware job creation, runtime config, and worker diagnostics.
- Python owns `group_memory_extract` and `rag_ingest` execution, group-memory extraction, RAGFlow upload/parse, checkpoint semantic content, and RAG strategy.

Tests run:

- `go test ./app/service -run "TestKnowledgeJobPlanner|TestKnowledgePipelineDiagnosticsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./cmd/agent-runtime -run "TestKnowledgeJobPlannerConfigFromEnv|TestRuntimeConfigFromEnvReportsSanitizedOneBotReadiness|TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`
- `uv run pytest tests/test_agent_gateway_knowledge_worker.py -q`
- `uv run pytest tests/test_bootstrap_wiring_p2.py -k "agent_runtime or agent_gateway" -q`

Findings:

- The first bootstrap targeted run exposed an existing `UnboundLocalError` when RAGFlow was disabled because `worker` was only assigned inside the RAGFlow-enabled branch. The worker construction was moved outside that branch.
- The planner is opt-in. Existing Python enqueue remains as compatibility fallback until `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED=true` is set on `agent-runtime`.

Decision:

- Accept. This moves stable recurring knowledge job admission to Go without moving AI extraction/RAG strategy out of Python.

Follow-ups:

- Live smoke: enable the planner, confirm `/v1/runtime-config` and `/v1/runtime-workers`, then verify Python no longer emits legacy enqueue logs while still processing leased jobs.
- Future queue cutover work should continue treating Python as the execution owner for knowledge AgentJobs.
