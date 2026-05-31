# Akashic Go 迁移 Backlog

最后更新：2026-05-31

本文件记录后续候选任务和长期规划。进入具体迭代前，先从这里挑选本轮能完成的事项写入 `docs/sdd/TODO.md`；本轮未承诺的事项不要放进 TODO。

## Go / Python 边界收敛

- 继续检查是否还有确定性 runtime 状态、幂等、调度、资产、队列、审计逻辑散落在 Python；能迁移则按 SDD 切片推进到 Go `services/agent-runtime`。
- 评估 proactive semantic items 是否继续留在 Python：它涉及语义候选、embedding/RAG 试错，原则上不急着迁 Go；如果迁移，也只考虑元数据、诊断或审计层。

## QQ / Telegram / Observe

- 推进 QQ/NapCat Go adapter cutover：先完成只读 readiness 和 live send smoke，再决定是否把 QQ channel alias 加入 Go outbound 或启用 Go local outbox worker。
- 继续梳理 Telegram/QQ inbound 接收与 Python agent 回复之间的边界，避免接收、发送、推理状态交叉堆在 Python。

## Queue / Worker / Runtime

- 设计并验证更合适的 MQ 方案，保留多线程消费和可替换边界，避免把 NATS/RabbitMQ/Redis Streams 细节泄漏到 domain。
- 继续推进 Python AI worker 作为 Go AgentJob consumer 的规范化：worker status 已有 lease/fencing/heartbeat renewal，external_lease 已有 ack/nack/term 执行诊断并进入 runtime overview；后续继续收敛 AgentJob lease、ack/fail、重试、外部 MQ result-ack live smoke 和切换门禁。

## Knowledge / Memory / RAG

- 继续把 group memory / RAG ingestion 的生命周期交给 Go job，Python 只做 AI worker 和策略实验。
- 后续评估 RAGFlow 或其它 RAG 组件与 Go runtime 的边界：Go 管任务、资产、索引状态和审计；Python 管 chunking、embedding、rerank、answer synthesis 实验。

## SDD / 文档治理

- 继续清理 SDD 文档结构：TODO 只记录本轮任务，DONE 记录已完成，LIVE_CHECKS 记录现场验证，BACKLOG 记录后续候选。
- 每次迭代都补 spec、review、测试命令和风险说明，并在结束前完成全部当前 TODO，避免代码先行或半成品切片导致结构再次变乱。
