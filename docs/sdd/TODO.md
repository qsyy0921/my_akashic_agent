# Akashic Go 迁移 TODO

最后更新：2026-06-02

## 使用规则

- 本文件只保留本轮迭代必须全部完成的短待办，原则上不超过 10 条。
- 每次迭代必须完成本文件列出的全部未完成项；不要把一个切片拆成“做一点就收工”的半成品。
- 如果某项本轮无法完成，必须在动手前移到 `BACKLOG.md`，或在本文件明确写阻塞原因、所需外部条件和恢复入口。
- 已完成迁移记录放到 `docs/sdd/DONE.md`。
- 所有待解决问题、风险、阻塞和待决策项先进入 `docs/sdd/PROBLEM_REGISTRY.md` / `docs/sdd/OPEN_ISSUES.md`，不要直接堆进本文件。
- 运行态观察、live smoke 和人工验证项放到 `docs/sdd/LIVE_CHECKS.md`。
- 长期规划、未来想法、非本轮任务放到 `docs/sdd/BACKLOG.md`，不要堆在本文件。
- 每轮结束前，本文件必须清空；只有真实外部阻塞才允许留下，并且必须写明阻塞原因。
- 每完成一轮迭代：更新本 TODO、补 SDD spec/review，并把完成项归档到 DONE。

## 本轮未完成

当前无未完成项。

## 边界约束

- Go 负责确定性基础设施：路由状态、幂等、持久化存储、生命周期、重试、租约、checkpoint、资产和审计。
- Python 负责 AI runtime：模型/供应商路由、prompt 和上下文管线、工具执行、RAG/Memory 抽取、chunking、检索策略、Embedding/Rerank/OCR/VLM、图片生成执行、群知识沉淀、评估脚本、策略实验和 provider-specific fallback。
- Python 可以持有算法/实验状态的本地 mirror，但一旦某类状态变成确定性控制面、审计、幂等、租约、调度、资产或队列生命周期，就应迁到 Go。
- 不为架构形式过度拆分服务；优先在 `services/agent-runtime` 内复用现有 DDD/六边形分层，只有当职责和部署边界真正独立时才新增服务。
- 每完成一轮迭代，都必须更新本中文 TODO，以及对应 SDD spec/review 文档；不要只提交一个未闭环切片。
