# Review: Phase 8.18 Knowledge Checkpoints

Spec:
- `docs/sdd/specs/agent-gateway/010-knowledge-checkpoints.md`

Implementation summary:
- Added a Go `KnowledgeCheckpoint` domain aggregate with non-regression
  cursor rules.
- Added app commands, queries, ports, assembler, and service for checkpoint
  get/upsert behavior.
- Added memory and JSON-file repositories, wired by
  `AKASHIC_KNOWLEDGE_CHECKPOINTS_DSN` or `AKASHIC_KNOWLEDGE_CHECKPOINTS_PATH`.
- Added `/v1/knowledge-checkpoints/{checkpoint_id}` HTTP get/upsert routes.
- Updated the Python agent runtime client and knowledge worker so `rag_ingest`
  reads the checkpoint before choosing `since_seq` and advances it after a
  successful non-empty RAGFlow ingest.
- Updated RAGFlow QQ group indexing output with `start_seq` and `end_seq`, and
  made empty selected-message batches succeed without advancing the cursor.

Tests run:
- `go test ./...`
- `python -m compileall integrations\agent_gateway.py integrations\agent_gateway_knowledge_worker.py agent\tools\ragflow.py`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_agent_gateway_knowledge_worker.py tests\test_agent_gateway_knowledge_worker_smoke.py tests\test_ragflow_integration.py -q --basetemp .tmp\pytest-knowledge-checkpoints`

Findings:
- The checkpoint state is infrastructure lifecycle state and fits Go
  `agent-runtime`; RAGFlow upload and parsing stays in Python.
- The cursor regression invariant is enforced in the domain model and covered
  by service tests.
- The smoke test required a fake checkpoint adapter so the worker exercises the
  same API boundary as live runtime.

Decision:
- Accept this slice. It moves RAG ingest replay state into Go without changing
  observe-only group collection semantics.

Follow-ups:
- Add dashboard visibility for checkpoint state once more RAGFlow datasets are
  configured.
- Consider a future generic checkpoint list API if multiple knowledge workers
  need operational inspection.
