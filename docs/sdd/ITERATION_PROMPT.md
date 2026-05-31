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
- 本轮结束前必须完成 TODO.md 中所有未完成项。
- 如果某项不能完成，必须是外部阻塞，并写清阻塞原因、所需条件和恢复入口；普通“还没做完”不能留在 TODO。
- 未来想法放 BACKLOG.md，现场观察放 LIVE_CHECKS.md，完成记录放 DONE.md。
- 每轮都要补 SDD spec/review、运行相关测试、更新中文 TODO/DONE/LIVE_CHECKS/BACKLOG。
- 每次修改后提交并推送到 GitHub。

架构边界：
- Go 只做确定性后端基础设施：路由、账号、幂等、防循环、inbox/outbox、media asset、job/queue、lease、retry、checkpoint、scheduler control plane、audit、dashboard diagnostics。
- Python 做 AI runtime：模型/provider 路由、prompt/context、工具执行、Memory/RAG 抽取与排序、embedding/rerank、OCR/VLM、图片生成执行、评估脚本、策略实验和 provider-specific fallback。
- Python 可以保留迁移期 mirror/fallback，但 Go 有对应 domain API 后，确定性基础设施状态必须以 Go 为权威。

当前原则：
- 不过度拆服务；优先在 services/agent-runtime 内按 DDD + 六边形分层演进。
- 不把 LLM 调用、prompt、RAG ranking、VLM/OCR prompt 或图片生成浏览器实现迁入 Go。
- 不破坏 observe-only QQ 群逻辑，不做未经 smoke 的真实平台发送 cutover。
```
