# Dashboard Receiver Statuses Table

## Goal

在 dashboard runtime-overview 中，把 `receiver_statuses` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`receiver_statuses` card detail 可直接看到：
   - receiver totals
   - 每个 receiver 的连接状态表
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不启动、停止或重连 receiver，不发送 QQ/Telegram，不触发 AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
