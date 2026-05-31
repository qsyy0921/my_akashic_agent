# Akashic Runtime Iteration Prompt

Use this prompt when resuming the Go/Python runtime migration in a new Codex
thread.

```text
继续 Akashic Go/Python runtime 迁移。

工作目录：E:\agent\akashic

先读取：
- git status --short
- docs/sdd/TODO.md
- docs/sdd/DONE.md
- docs/sdd/LIVE_CHECKS.md
- docs/sdd/BACKLOG.md
- 本轮相关 SDD spec/review

迭代规则：
- docs/sdd/TODO.md 是本轮必须闭环的契约，不是长期愿望清单。
- 本轮结束前必须完成 TODO.md 中所有未完成项；不要只完成其中一个切片就停止，也不要把“下一轮继续同一 TODO”当成交付。
- 每次迭代的交付口径是“当前 TODO 全量清零”：如果发现 TODO 太大，必须先在 SDD 中重新收敛范围，把非本轮事项移到 BACKLOG.md，再开始写代码。
- 如果某项不能完成，必须是外部阻塞，并写清阻塞原因、所需条件和恢复入口；普通“还没做完”不能留在 TODO。
- 如果用户在迭代中追加本轮必须完成的要求，先把它写进 TODO.md，再和原 TODO 一起全量闭环。
- 未来想法放 BACKLOG.md，现场观察放 LIVE_CHECKS.md，完成记录放 DONE.md。
- 每轮都要补 SDD spec/review、运行相关测试、更新中文 TODO/DONE/LIVE_CHECKS/BACKLOG。
- 每次修改后提交并推送到 GitHub。

架构边界：
- Go 只做确定性后端基础设施：路由、账号、幂等、防循环、inbox/outbox、media asset、job/queue、lease、retry、checkpoint、scheduler control plane、audit、dashboard diagnostics。
- Python 做 AI runtime：模型/provider 路由、prompt 模板、上下文压缩、对话状态构造、工具调用编排、Memory/RAG 抽取与合并、chunking、embedding/rerank、retrieval/ranking 策略、OCR/VLM 图片理解、图片生成执行、浏览器/第三方 AI 工具适配、群知识沉淀、攻略/FAQ 总结、评估脚本、离线实验和 provider-specific fallback。
- Python 可以做算法试错和非确定性推理编排；Go 不接管 prompt、LLM/VLM/OCR 调用、RAG ranking、图片生成浏览器控制或模型供应商 fallback。
- Python worker 通过 Go domain API 获取 lease、回写结果、上报 heartbeat/status、读取 checkpoint；它不拥有队列生命周期、幂等、防循环、资产访问控制或审计权威状态。
- Python 可以保留迁移期 mirror/fallback，但 Go 有对应 domain API 后，确定性基础设施状态必须以 Go 为权威，Python mirror 只能用于兼容读取或离线实验。

当前原则：
- 不过度拆服务；优先在 services/agent-runtime 内按 DDD + 六边形分层演进。
- 不把 LLM 调用、prompt、RAG ranking、VLM/OCR prompt 或图片生成浏览器实现迁入 Go。
- 不破坏 observe-only QQ 群逻辑，不做未经 smoke 的真实平台发送 cutover。
- 每轮都要以“完成当前 TODO 全量内容”为交付单位；需要缩小范围时先改 TODO/SDD，不在实现后留下半成品或“下一轮继续同一 TODO”的状态。
```
