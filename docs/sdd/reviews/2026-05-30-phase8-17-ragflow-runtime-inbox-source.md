# Review: RAGFlow Runtime Inbox Source

Spec:
- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/specs/agent-gateway/009-inbox-raw-message-store.md`
- `docs/sdd/specs/group-message-memory/001-processing-pipeline.md`

Implementation summary:
- Extended `RAGFlowIndexQQGroupTool` to accept the same group-message source
  port used by group memory.
- Wired agent-runtime knowledge worker RAGFlow indexing to use
  `AgentRuntimeInboxGroupMessageSource`, so `rag_ingest` jobs read Go
  `/v1/inbox` instead of Python session internals when runtime is enabled.
- Kept the Python session-store source as the fallback for direct/manual tools
  when `agent_runtime` is disabled.
- Raised Go inbox replay limits to 5000 so RAG ingest can honor its existing
  `max_messages` contract.

Tests run:
- `python -m compileall agent\tools\ragflow.py bootstrap\app.py bootstrap\tools.py integrations\agent_runtime_inbox_source.py group_memory\sources.py`
- `uv run pytest tests\test_ragflow_integration.py tests\test_agent_runtime_inbox_source.py tests\test_bootstrap_wiring_p2.py tests\test_agent_gateway_knowledge_worker.py tests\test_agent_gateway_knowledge_worker_smoke.py -q --basetemp .tmp\pytest-ragflow-runtime-inbox`
- `go test ./...` from `services/agent-runtime`

Findings:
- RAGFlow ingest still performs upload/parse work in Python because it is an AI
  integration/tool boundary, not durable runtime infrastructure.
- Go now owns the observed group message replay source for both group memory and
  RAGFlow ingest.

Decision:
- Proceed. This keeps the Go/Python split aligned: Go owns durable event replay;
  Python owns RAGFlow adapter behavior and document formatting.

Follow-ups:
- Add an explicit RAG ingest cursor/checkpoint model if repeated uploads become
  noisy.
- Include media asset ids in uploaded RAGFlow documents once citations are
  standardized.
