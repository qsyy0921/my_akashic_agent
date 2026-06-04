# 2026-06-03 Phase8-280 Dashboard Knowledge Pipelines Top-level Parity

## 结论

已完成。

## 本轮交付

- 修复 `/api/dashboard/runtime-overview` normalize 路径遗漏的 top-level
  `knowledge_pipelines`
- 补 regression test，覆盖 top-level totals parity
- 重新验证 dashboard knowledge/RAG live boundary

## 已验证

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_dashboard_knowledge_rag_state_boundary.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## 当前 live 结果

- dashboard top-level `knowledge_pipelines` 与 runtime card/detail 一致
- `dashboard_knowledge_rag_state_boundary` 恢复 `live_verified`

## 风险边界

- 仅修复 dashboard payload parity
- 不修改 knowledge planner admission、Python worker 执行或 RAG dataset 配置
