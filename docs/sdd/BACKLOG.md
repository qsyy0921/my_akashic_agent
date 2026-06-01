# Akashic Go 迁移 Backlog

最后更新：2026-06-01

本文件记录后续候选任务和长期规划。进入具体迭代前，先从这里挑选本轮能完成的事项写入 `docs/sdd/TODO.md`；本轮未承诺的事项不要放进 TODO。

## Go / Python 边界收敛

- 继续检查是否还有确定性 runtime 状态、幂等、调度、资产、队列、审计逻辑散落在 Python；能迁移则按 SDD 切片推进到 Go `services/agent-runtime`。
- 评估 proactive semantic items 是否继续留在 Python：它涉及语义候选、embedding/RAG 试错，原则上不急着迁 Go；如果迁移，也只考虑元数据、诊断或审计层。

## QQ / Telegram / Observe

- 推进 QQ/NapCat Go adapter cutover：只读 `/v1/outbound-cutover/readiness`、`/v1/outbound-cutover/plan` 和 runtime overview 聚合已完成；后续先做真实 live send smoke，再决定是否把 QQ channel alias 加入 Go outbound 或启用 Go local outbox worker。
- 继续梳理 Telegram/QQ inbound 接收与 Python agent 回复之间的边界，避免接收、发送、推理状态交叉堆在 Python。
- media asset content diagnostics 已接入 runtime overview，Python dashboard 已规范化该 detail；若前端需要更细 dashboard drilldown，可直接消费 `media_asset_content_diagnostics`，但不要在 Go 或 dashboard 中加入 OCR/VLM/文件语义解析。

## Queue / Worker / Runtime

- 基于已完成的 local worker 与 NATS `external_lease` 账号级节流，后续再评估持久化/分布式 rate-limit state、全局 backpressure、平台风控策略参数化和生产 cutover；切换前必须保持可回滚和 observe-only smoke。
- 生产 cutover 前继续做真实 NATS external lease preflight：确认 local outbox worker 已关闭、delivery smoke readiness 与 runtime overview `Delivery Smoke` card 均通过、dual-read smoke 已通过、outbox external lease smoke 已通过，再启用真实平台发送范围。
- MQ 方案下一步只做有证据的 adapter 扩展：NATS JetStream 已是当前推荐外部 MQ 且 provider capability matrix 已可见；Redis Streams / RabbitMQ 仅在出现明确部署需求时再实现 infrastructure adapter，并继续保持 domain/provider-neutral。
- 若前端需要更强队列可视化，再把 provider capability 做成独立 dashboard 表格；当前 runtime overview 已提供 summary/card 级摘要，避免本轮扩大 UI 改造范围。
- queue topology read model 已由 Go 提供 `/v1/queue-topology`，并已聚合到 runtime overview / Python dashboard；若前端需要更强执行边界可视化，可基于该 detail 做独立面板，但不要在 dashboard 中加入真实控制逻辑。
- 继续推进 Python AI worker 作为 Go AgentJob consumer 的规范化：worker status 已有 lease/fencing/heartbeat renewal，external_lease 已有 ack/nack/term 执行诊断并进入 runtime overview，`agent_job` result-ack readiness/plan 及二者 runtime overview 聚合已可见；后续继续收敛外部 MQ result-ack live smoke 和切换后的稳定性观察。
- 若后续要做 Python worker 并发控制、autoscaling 或知识任务优先级调度，优先基于已落地的 Go `AgentJob` pressure、`Agent Job Worker Coverage`、`/v1/agent-job-capacity/plan`、`/v1/agent-job-priority/plan` 和 Go-owned `Operator Approval Ledger`；当前只提供只读容量/优先级建议与 operator 审计记录，不引入调度副作用。
- `Agent Job Capacity` 与 `Agent Job Priority` 已进入 runtime overview 且 Python dashboard 已规范化 detail；后续如需前端 drilldown，可直接消费 dashboard detail，真实 autoscaling/优先级控制仍必须另做 operator ack 和审计设计。
- 后续若要把 capacity / priority / cutover plan 推进成真实控制面，已有 operator approval ledger 可作为人工确认记录；下一步仍必须把具体控制面 mutation 绑定到 approval id、配置变更审计、限流/熔断策略、回滚记录和 Python worker 侧并发实现。不要直接让 Go 自动启动 AI worker 或修改模型/RAG/图片执行策略。

## Knowledge / Memory / RAG

- 若前端需要更完整的知识任务启用前可视化门禁，可把 runtime overview 中的 `Knowledge Planner` / `Knowledge Planner Readiness` cards 做成 dashboard drilldown；当前已提供稳定 Go API 和 aggregate，避免扩大 UI 范围。
- knowledge planner cutover plan 已提供只读 enable/verify/rollback 步骤，Python dashboard 已规范化该 detail；若后续需要自动化 planner cutover，必须将 mutation 显式绑定到 Go operator approval ledger 的有效记录、配置变更审计和回滚记录，不能让 plan endpoint 或 dashboard 自动改环境变量或启动 worker。
- group memory / RAG ingestion 的 AgentJob 生命周期与定期 admission 已交给 Go；后续只在出现明确需求时再推进优先级、并发控制、dataset/index readiness 或外部 MQ result-ack 的 live cutover，Python 继续做 AI worker 和策略实验。
- 当前 source-seq lag、checkpoint age/stagnant、per-group stage lease freshness 和 per-dataset RAG state 已落地；后续若要把群知识编排再往前推进，可继续增加更接近外部索引的 dataset/index state，但先保持只读控制面。
- 当前配置期 dataset 绑定、成功 `rag_ingest` 快照和 derived `rag_index_state` 已纳入 Go 只读 control-plane；后续若继续推进，可考虑接入更稳定的外部 index parse 状态或文档总量，但仍应避免把 provider-specific 原始 payload 直接耦合进 Go。
- 当前 source-seq lag 已落地；后续若继续推进，可考虑把 lag 与更稳定的 dataset/index metadata 关联，但前提是这些状态先成为 Go-owned 控制面，而不是直接侵入 Python RAGFlow 实验逻辑。
- 后续评估 RAGFlow 或其它 RAG 组件与 Go runtime 的边界：Go 管任务、资产、索引状态和审计；Python 管 chunking、embedding、rerank、answer synthesis 实验。

## SDD / 文档治理

- 继续清理 SDD 文档结构：TODO 只记录本轮任务，DONE 记录已完成，LIVE_CHECKS 记录现场验证，BACKLOG 记录后续候选。
- 每次迭代都补 spec、review、测试命令和风险说明，并在结束前完成全部当前 TODO，避免代码先行或半成品切片导致结构再次变乱。
