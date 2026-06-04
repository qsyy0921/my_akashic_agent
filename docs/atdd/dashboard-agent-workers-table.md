# Dashboard Agent Workers Table

## Goal

在 dashboard runtime-overview 中，把 `agent_workers` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`agent_workers` card detail 可直接看到：
   - worker totals
   - 每个 worker 的状态、lease、stale 和 reason
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不启动/停止/替换 worker，不修改 lease，不发送 QQ/Telegram，不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
