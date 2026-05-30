# Review: Phase 8.20 Knowledge Checkpoint Dashboard

Spec:
- `docs/sdd/specs/agent-gateway/010-knowledge-checkpoints.md`

Implementation summary:
- Added Go `ListKnowledgeCheckpoints` repository/query path and
  `GET /v1/knowledge-checkpoints`.
- Added a Python dashboard plugin that reads the Go runtime API and provides
  pageable/filterable checkpoint visibility.
- Added a dashboard panel for checkpoint id, cursor, group id, dataset id,
  source, update time, and metadata detail.

Tests run:
- `go test ./...`
- `python -m compileall plugins\knowledge_checkpoints\dashboard.py`
- `npm run build:plugins`
- `uv run pytest tests\test_knowledge_checkpoints_dashboard_plugin.py tests\test_agent_gateway_client.py tests\test_agent_gateway_knowledge_worker.py tests\test_agent_gateway_knowledge_worker_smoke.py tests\test_ragflow_integration.py -q --basetemp .tmp\pytest-knowledge-dashboard`

Findings:
- Dashboard visibility needs a Go list API; reading the JSON file directly
  would bypass the runtime boundary and weaken the DDD/hexagonal split.
- Python remains an HTTP adapter for dashboard presentation only.

Decision:
- Accept the design. Go owns checkpoint state and query behavior; Python
  dashboard code only presents the runtime-owned state.

Follow-ups:
- Add checkpoint lag diagnostics once RAGFlow datasets are actively configured.
