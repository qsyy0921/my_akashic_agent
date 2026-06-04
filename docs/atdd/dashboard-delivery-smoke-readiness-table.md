# Dashboard Delivery Smoke Readiness Table

## Goal

在 dashboard runtime-overview 中，把 `delivery_smoke_readiness` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`delivery_smoke` card detail 可直接看到：
   - `ready`
   - `cases/ready/not_ready`
   - smoke case table
   - notes
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不发送 QQ/Telegram 消息，不创建 outbox，不触发 AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
