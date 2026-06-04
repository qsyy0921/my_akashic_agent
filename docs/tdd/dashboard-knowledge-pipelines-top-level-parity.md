# Dashboard Knowledge Pipelines Top-level Parity TDD

## Python

1. `plugins/runtime_overview/dashboard.py`
   - normalize runtime-overview payload 时返回 top-level
     `knowledge_pipelines`
2. `tests/test_runtime_overview_dashboard_plugin.py`
   - fallback / normalize case 断言 top-level `knowledge_pipelines` totals
3. `tests/test_verify_dashboard_knowledge_rag_state_boundary.py`
   - live verifier shape 继续要求 dashboard top-level parity 为 true
