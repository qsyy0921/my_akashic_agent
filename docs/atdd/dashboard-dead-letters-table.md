# ATDD - Dashboard Dead Letters Table

## Scenario

当 operator 打开 runtime overview dashboard 并查看 `dead_letters` detail 时，应能
直接看到 AgentJob 和 outbox 的 dead-letter 聚合结果，而不是只能读取 raw JSON。

## Acceptance

1. panel 显示 `Dead Letters`。
2. panel 显示 `No agent-job dead letters sampled`。
3. panel 显示 `No outbox dead letters sampled`。
4. panel 显示 `Agent Job Dead Letter Totals`。
5. unified goal verifier 返回：
   - `dashboard_read_models.dead_letters_table=true`
