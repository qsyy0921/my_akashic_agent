# TDD - Dashboard RAG Eval Failures Table

## Tests

- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言 panel JS 资产包含：
  - `RAG Eval Failures`
  - `No rag-eval failures sampled`
  - `rag eval dead letters`
  - `Quality`
- 在 `tests/test_runtime_overview_dashboard_plugin.py` 断言 runtime overview
  payload 中的 `rag_eval_failures` card 保留结构化 detail。
- 继续运行 SDD governance/spec index 测试，确保新 spec 已被索引。
