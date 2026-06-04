# Dashboard Agent Job Pressure Table

## Goal

在 dashboard runtime-overview 中，把 `agent_job_pressure` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`agent_job_pressure` card detail 可直接看到：
   - 当前 pressure summary
   - throughput / dead-letter 摘要
   - by-type pressure rows
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不创建/租约/执行 AgentJob，不修改 queue，不发送 QQ/Telegram，不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
