# 217 Dashboard Knowledge RAG State Boundary Live Verifier

最后更新：2026-06-03

## 问题

当前仓库已经有 Go-owned `knowledge_pipelines` control plane：

- `/v1/knowledge-pipeline-diagnostics`
- `/v1/runtime-overview` `knowledge_pipelines` card/detail
- dashboard `/api/dashboard/runtime-overview` fallback/read-model

但在本轮之前，unified goal verifier 对
`dashboard_read_models.knowledge_pipelines_table` 的证据仍偏弱：

1. 主要依赖 panel 静态字符串，而不是当前 turn 的 live parity；
2. dashboard fallback 没有稳定补齐 `knowledge_pipelines` top-level payload；
3. 默认 timeout 偏小，live dashboard 容易把 knowledge/RAG detail 退化成空态。

## 目标

新增一个 repo-owned live verifier，直接比较三层当前 turn 证据：

- Go `/v1/knowledge-pipeline-diagnostics`
- Go `/v1/runtime-overview`
- dashboard `/api/dashboard/runtime-overview`

并把当前 checkpoint-derived knowledge/RAG state 固定进 unified verifier。

## 非目标

- 不直接调用外部 RAGFlow 或 index service
- 不创建 checkpoint、不创建或执行 AgentJob
- 不修改 observe target、receiver、worker、queue 或 dashboard SQLite
- 不把 checkpoint snapshot 推导误报为外部 index 已 live 验证

## 设计

### 入口

- Python verifier:
  `scripts/verify_dashboard_knowledge_rag_state_boundary.py`
- PowerShell wrapper:
  `scripts/verify-dashboard-knowledge-rag-state-boundary.ps1`

### dashboard fallback 补强

`plugins/runtime_overview/dashboard.py` 必须：

1. fallback 读取 `/v1/knowledge-pipeline-diagnostics`
2. 归一化 `knowledge_pipelines` totals / notes / pipeline rows
3. 在 Go runtime-overview 不完整时补：
   - top-level `knowledge_pipelines`
   - `knowledge_pipelines` summary fields
   - `knowledge_pipelines` card
4. 对 diagnostics fallback 使用不小于 5 秒的读取超时，避免当前 live
   dashboard 因默认超时太短而退化成空 detail

### verifier 必须验证的边界

1. direct runtime endpoint 暴露 `knowledge_pipelines` targets
2. direct runtime endpoint 暴露 RAG boundary fields：
   - `configured_rag_datasets`
   - `rag_datasets`
   - `rag_dataset_index_ready`
   - `rag_dataset_index_missing_snapshot`
   - `rag_dataset_index_empty`
   - `rag_dataset_index_lagging`
3. runtime overview card 与 direct endpoint 的 totals/card value/status 一致
4. dashboard card 与 runtime overview card 一致
5. dashboard top-level `knowledge_pipelines` 与 runtime overview detail 一致
6. 当前边界明确保持 `side_effect=none`

### Unified verifier 接入

`scripts/verify-go-migration-goal.ps1` 必须接入：

- 顶层 `dashboard_knowledge_rag_state_boundary`
- `residual_classification.knowledge_rag_state_boundary`
- `migration_residuals.knowledge_rag_state_boundary`
- `migration_bucket_summary.already_in_go_only_missing_live_verification`

同时：

- `dashboard_read_models.knowledge_pipelines_table`
  不再只依赖 panel 字符串
- `current_state` 需要补：
  - `knowledge_configured_rag_datasets`
  - `knowledge_rag_datasets`
  - `knowledge_rag_dataset_index_ready`

## 验收

1. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-knowledge-rag-state-boundary.ps1`
   返回 `conclusion.status=live_verified`
2. dashboard `/api/dashboard/runtime-overview` 当前 turn 暴露：
   - `knowledge_pipelines` card
   - top-level `knowledge_pipelines`
3. unified verifier 当前 turn 输出：
   - `dashboard_read_models.knowledge_pipelines_table=true`
   - `residual_classification.knowledge_rag_state_boundary.category=checkpoint_snapshot_derived_control_plane_live_verified`
4. 当前无 configured datasets 时，verifier 结论允许保持
   `dashboard_knowledge_rag_state_boundary_live_verified_without_configured_datasets`
   而不是误报失败
