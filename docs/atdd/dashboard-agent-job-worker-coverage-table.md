# Dashboard Agent Job Worker Coverage Table

## Goal

在 dashboard runtime-overview 中，把 `agent_job_worker_coverage` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`agent_job_worker_coverage` card detail 可直接看到：
   - 每个 job type 的 expected worker types
   - 当前 active/running/failed/stale/high_pressure 与 coverage 状态
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不启动/停止/替换 worker，不创建/租约/执行 AgentJob，不发送 QQ/Telegram，不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
