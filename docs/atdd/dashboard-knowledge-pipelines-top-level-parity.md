# Dashboard Knowledge Pipelines Top-level Parity ATDD

## 场景

当 dashboard 读取 Go runtime-overview 的 `knowledge_pipelines` card/detail
时，top-level `knowledge_pipelines` 也必须返回同一份只读数据。

## 验收步骤

1. 调用 `GET /v1/runtime-overview`
2. 调用 `GET /api/dashboard/runtime-overview`
3. 对比：
   - runtime card `knowledge_pipelines.detail.knowledge_pipelines`
   - dashboard card `knowledge_pipelines.detail.knowledge_pipelines`
   - dashboard top-level `knowledge_pipelines`
4. 断言三者的 `totals/targets/notes/side_effect` 一致
