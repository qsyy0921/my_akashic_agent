# TDD: Dashboard Knowledge RAG State Boundary Live Verifier

## Tests

1. Add focused tests for
   `scripts/verify_dashboard_knowledge_rag_state_boundary.py` covering:
   - live-verified parity shape
   - missing dashboard card failure
2. Extend `tests/test_runtime_overview_dashboard_plugin.py` to assert fallback
   synthesis of:
   - top-level `knowledge_pipelines`
   - `knowledge_pipelines` summary fields
   - `knowledge_pipelines` card
3. Extend `tests/test_verify_go_migration_goal_script.py` to assert:
   - the new PS1 wrapper path is wired
   - the verifier is invoked
   - current-state knowledge/RAG fields are surfaced
   - residual classification wiring exists
4. Rerun the focused verifier tests, dashboard plugin tests, and the unified
   goal verifier end to end.
