# Dashboard Knowledge Pipelines Table

## Goal

在 dashboard runtime-overview 中，把 `knowledge_pipelines` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`knowledge_pipelines` card detail 可直接看到：
   - pipeline totals
   - 每个 target 的 capture / group_memory / rag_ingest / coverage 状态
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不创建 checkpoint、不创建或执行 AgentJob、不触发 QQ/Telegram、不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
