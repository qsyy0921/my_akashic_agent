# 223. Dashboard Knowledge Pipelines Top-level Parity

## 背景

`/api/dashboard/runtime-overview` 已能正确返回 `knowledge_pipelines` card 和
detail，但 runtime-overview normalize 路径遗漏了 top-level
`knowledge_pipelines` payload，导致 live verifier 把 dashboard 降级成
partial。

## 目标

1. normalize 路径返回 top-level `knowledge_pipelines`
2. top-level payload 与 card/detail 使用同一份 runtime data
3. `verify-dashboard-knowledge-rag-state-boundary.ps1` 恢复 `live_verified`

## 非目标

- 不修改 knowledge planner admission
- 不修改 RAG dataset/index runtime state
- 不修改 Python knowledge worker 执行语义

## 设计

### 1. 补齐 normalize 返回值

`_normalize_runtime_overview_payload()` 返回体必须包含：

- `knowledge_pipelines`

其值直接复用已归一化后的 `knowledge_pipelines` 对象。

### 2. Regression test

`tests/test_runtime_overview_dashboard_plugin.py` 需要显式断言：

- `payload["knowledge_pipelines"]["totals"]["targets"]`
- `payload["knowledge_pipelines"]["totals"]["rag_dataset_index_missing_snapshot"]`

## 验收

1. `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_dashboard_knowledge_rag_state_boundary.py ...`
   通过
2. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
   返回 `conclusion.status=live_verified`
3. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
   恢复：
   - `dashboard_read_models.knowledge_pipelines_table=true`
   - `current_state.dashboard_fallback_category=live_verified_runtime_read_models`
