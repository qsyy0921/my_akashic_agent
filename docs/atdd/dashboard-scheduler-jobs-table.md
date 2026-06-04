# Dashboard Scheduler Jobs Table

## Goal

在 dashboard runtime-overview 中，把 `scheduler_jobs` 从 raw JSON 提升为结构化只读 drilldown。

## Acceptance

1. 打开 runtime overview panel 时，`scheduler_jobs` card detail 可直接看到：
   - scheduler totals
   - trigger/tier/status 分布
   - recent job sample rows
2. detail 打开后仍保留 raw JSON fallback。
3. 渲染 detail 不创建/修改/删除 scheduler job，不执行 complete/recovery，不发送 QQ/Telegram，不触发 Python AI。
4. unified goal verifier 能在当前轮次证明 dashboard panel 资产已包含该结构化 read-model。
