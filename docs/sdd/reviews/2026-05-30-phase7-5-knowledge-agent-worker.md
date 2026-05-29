# Review: Phase 7.5 Knowledge Agent Worker

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`

Implementation summary:

- Added `AgentGatewayKnowledgeWorker` to enqueue observe-only QQ group
  `group_memory_extract` jobs and optional RAGFlow `rag_ingest` jobs through
  the Go `agent-gateway`.
- The worker leases `group_memory_extract` and `rag_ingest` from `/v1/jobs`,
  executes the existing `GroupMemoryService` and `RAGFlowIndexQQGroupTool`, and
  completes/fails the generic jobs.
- App startup now skips the old direct `GroupMemoryLoop` when
  `integrations.agent_gateway.enabled=true`, preventing duplicate in-process
  ingestion.
- Added `knowledge_job_interval_seconds` to the agent gateway integration
  config.
- Added tests for enqueue, group-memory execution, RAG ingest execution,
  RAGFlow failure handling, and idle no-job behavior.

Tests run:

- Implemented checks executed in this phase:
  - `uv run pytest tests/test_agent_gateway_knowledge_worker.py tests/test_agent_gateway_client.py tests/test_agent_gateway_image_worker.py tests/test_bootstrap_wiring_p2.py tests/test_sdd_contract_fixtures.py tests/test_shadow_gateway.py -q --basetemp .tmp\\pytest-phase7-5-run`
    (48 passed)
  - `python -m compileall integrations\\agent_gateway_knowledge_worker.py bootstrap\\app.py agent\\config.py agent\\config_models.py config.example.toml`
  - `go test ./...` under `services/agent-gateway`
  - `go build ./cmd/agent-gateway` under `services/agent-gateway`
  - live smoke: group_memory_extract via real `GroupMemoryService` -> `succeeded`, rag_ingest via fake indexer -> `succeeded`
  - `git diff --check` (passed, only CRLF normalization warnings)

Findings:

- This keeps observe-only safety: jobs operate on already-observed sessions and
  never emit group-visible replies.
- RAG ingestion is enabled only when RAGFlow is configured with API key and at
  least one default dataset id.

Decision:

- Accept as the group-memory/RAG observe-only generic-job migration slice.

Follow-ups:

- Add durable job persistence before relying on jobs across gateway restarts.
- Add dashboard job panel for group-memory/RAG job visibility.
- Add platform delivery rules for completed image jobs that need direct replies.
