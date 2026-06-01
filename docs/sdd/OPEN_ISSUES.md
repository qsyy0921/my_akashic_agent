# Akashic Go 迁移 Open Issues

最后更新：2026-06-01

本文件是“所有待解决问题”的总账，和其它 SDD 文档分工如下：

- `TODO.md`：只放本轮必须全部完成的短任务，结束前必须清空或写明阻塞。
- `DONE.md`：只放已经完成并验证的事实。
- `BACKLOG.md`：放未来候选任务、长期规划和暂不承诺的改造方向。
- `LIVE_CHECKS.md`：放需要真实运行环境或人工观察的验证项。
- `OPEN_ISSUES.md`：放尚未解决的问题、风险、待决策项和需要后续跟踪的缺口。

进入某轮迭代前，如果某个 open issue 要在本轮解决，先把它拆成可完成任务写入 `TODO.md`；完成后从本文件移除或改为已解决记录并同步 `DONE.md`。

## 当前未解决问题

| ID | 领域 | 问题 | 影响 | 下一步 | 状态 |
| --- | --- | --- | --- | --- | --- |
| OI-001 | QQ / Telegram / Observe | QQ/NapCat Go adapter cutover 仍未进入真实 live send 范围。 | Go 已有 readiness/plan，但真实发送边界仍需人工 smoke 后才能切换。 | 按 `LIVE_CHECKS.md` 做 delivery smoke、dual-read、external lease preflight，再决定是否启用 Go local outbox worker 或 Go outbound channel alias。 | Open |
| OI-002 | Queue / MQ | NATS JetStream 是当前推荐 MQ，但 Redis Streams / RabbitMQ adapter 尚未实现。 | 当前已保留 provider-neutral 边界；如果部署环境指定其它 MQ，需要新增 adapter。 | 只有出现明确部署需求时再做 infrastructure adapter，不提前扩大实现面。 | Open |
| OI-003 | AgentJob / Worker | `agent_job` external lease result-ack 仍处于 readiness/plan/smoke 边界。 | Python AI worker 继续执行 job；Go 还没有进入生产 result-ack cutover。 | 完成 live smoke，确认 strict lease token、worker coverage、rollback 后再启用。 | Open |
| OI-004 | Control Plane | capacity / priority / cutover plan 还没有真实 mutation executor。 | 目前只能给只读建议和审计，不能自动调并发、启动 worker 或修改配置。 | 后续必须绑定 operator approval、control mutation audit、allowlist、回滚和限流策略后再实现。 | Open |
| OI-005 | Media Assets | 媒体内容访问已可诊断、代理，并新增只读 recovery plan，但远程重新下载、文件恢复和内容缓存 executor 仍未统一。 | 图片/附件本地缺失时可看到恢复步骤和 future executor scope，但不能自动修复。 | 设计独立 media downloader/cache SDD；Go 管资产生命周期和 operator-approved executor，Python 仍管 OCR/VLM/语义解析。 | Open |
| OI-006 | Knowledge / RAG | RAG dataset/index state 目前主要来自 Go checkpoint snapshot 推导。 | 不能直接证明外部 RAGFlow/索引服务的真实 parse/index 状态。 | 若需要更强可靠性，接入稳定外部 index metadata，但不要把 provider-specific 原始 payload 耦合进 Go。 | Open |
| OI-007 | Python AI Worker | Python worker 并发控制、autoscaling 和优先级调度还没有真实控制面执行器。 | 大量 image / knowledge / rag job 堆积时，Go 只能诊断和建议。 | 基于 AgentJob pressure、worker coverage、capacity/priority plan 设计 operator-approved executor。 | Open |
| OI-008 | Frontend / Dashboard | runtime overview 已提供大量 card/detail，media asset content 和 queue topology 已有表格化 drilldown，但部分其它 drilldown 仍依赖原始 JSON。 | 运维可读性仍有不足，复杂状态需要人工读 payload。 | 优先消费现有 Go overview detail 做只读表格，不在 dashboard 加真实控制逻辑。 | Open |
| OI-009 | SDD Governance | 需要持续防止 TODO、BACKLOG、LIVE_CHECKS、OPEN_ISSUES 职责混用。 | 文档漂移会导致迭代再次变成半成品堆积。 | 每轮结束前同步检查四类文档；新增 open issue 时写入本文件而不是 TODO。 | Open |
