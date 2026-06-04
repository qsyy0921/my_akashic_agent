# Dashboard Runtime Workers Table

## Goal

在 dashboard runtime-overview 中，把 `runtime_workers` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`runtime_workers` card detail 可直接看到：
   - totals
   - 每个 Go worker 的 enabled/running 与 interval/batch 边界
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不启动/停止/重启 worker，不修改 queue flags，不发送 QQ/Telegram，不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
