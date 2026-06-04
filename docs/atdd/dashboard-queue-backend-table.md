# Dashboard Queue Backend Table

## Goal

在 dashboard runtime-overview 中，把 `queue_backend` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`queue_backend` card detail 可直接看到：
   - provider/mode
   - outbox owner/scope
   - agent_job owner
   - recommended backend
   - provider capability matrix
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不连接 MQ、不 publish/lease/ack/nack/term、不发送 QQ/Telegram、不触发 AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
