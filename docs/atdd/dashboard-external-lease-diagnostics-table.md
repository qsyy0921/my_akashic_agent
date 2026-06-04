# Dashboard External Lease Diagnostics Table

## Goal

在 dashboard runtime-overview 中，把 `external_lease_diagnostics` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`external_lease_diagnostics` card detail 可直接看到：
   - 当前 provider/mode 和外部队列开关
   - selected provider 是否支持 external lease / result-ack
   - outbox / agent_job execution owner
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不修改 MQ、不 publish/lease/ack/nack/term、不创建或执行 AgentJob/outbox、不触发 QQ/Telegram、不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
