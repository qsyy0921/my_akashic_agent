# Akashic 项目状态总览

最后更新：2026-06-02

本文件是项目状态总账，用来在重启 Codex、人工交接或开始新一轮迭代前快速判断“现在做到哪、还有什么坑、下一步从哪里进”。它不替代其它 SDD 文档，而是把它们的职责汇总到一个入口。

## 文档分工

| 文档 | 作用 | 使用规则 |
| --- | --- | --- |
| `TODO.md` | 本轮迭代必须全部完成的短待办 | 每轮结束前必须清空，或写明真实外部阻塞 |
| `DONE.md` | 已完成并通过检查的事实归档 | 只写已经落地的能力，不写计划 |
| `OPEN_ISSUES.md` | 所有未解决问题、风险、待决策项 | 新发现的问题先写这里，不要堆进 TODO |
| `BACKLOG.md` | 长期规划、候选任务、未来可做事项 | 进入迭代前再拆到 TODO |
| `LIVE_CHECKS.md` | 需要真实运行环境验证的检查项 | 不是开发任务，不要回填到 TODO |
| `specs/` | 每个能力的 SDD 设计契约 | 先写设计，再改代码 |
| `reviews/` | 每轮代码/架构审查记录 | 记录边界、测试和残余风险 |

## 当前迭代状态

当前 `TODO.md` 已清空。最近完成的切片是 dashboard media content recovery table：

- Runtime overview dashboard 已把 `media_asset_content_recovery` detail 从纯 JSON 提升为只读表格。
- 该表格展示 recovery audit totals、plan/preflight/recovery links 和 recent mutation rows；渲染时不调用 preflight/recovery，不创建 approval/mutation，不下载/缓存内容，不触发 Python AI。

对应设计：

- `docs/sdd/specs/agent-gateway/136-dashboard-media-content-recovery-table.md`

## 当前已完成主线

- Go 服务已统一为 `services/agent-runtime`，定位为 Agent Runtime / Control Plane，而不是单纯 gateway。
- Go 负责确定性基础设施：消息接入、去重、发送日志、防循环、队列生命周期、租约、checkpoint、media asset、审计、dashboard diagnostics。
- Python 负责 AI runtime：模型调用、prompt、工具执行、Memory/RAG/OCR/VLM、图片生成、群知识沉淀和策略实验。
- QQ / Telegram / observe-only、media asset diagnostics、runtime overview、AgentJob、queue topology、operator approval、control mutation audit 等基础能力已经逐步落地。
- SDD 文档已拆分为 TODO / DONE / OPEN_ISSUES / BACKLOG / LIVE_CHECKS，避免 TODO 无限膨胀。

完整完成记录见：

- `docs/sdd/DONE.md`
- `docs/sdd/reviews/README.md`

## 当前未解决问题

以下是 `OPEN_ISSUES.md` 的高层摘要，详细字段和下一步以 `OPEN_ISSUES.md` 为准。

| ID | 领域 | 未解决点 | 当前处理原则 |
| --- | --- | --- | --- |
| OI-001 | QQ / Telegram / Observe | QQ/NapCat Go adapter cutover 还没进入真实 live send 范围 | 先做 live smoke，再决定是否切换真实发送 |
| OI-002 | Queue / MQ | NATS JetStream 是当前推荐 MQ，Redis Streams / RabbitMQ adapter 尚未实现 | 只有出现明确部署需求才新增 adapter |
| OI-003 | AgentJob / Worker | `agent_job` external lease result-ack 仍在 readiness/plan/smoke 边界 | 完成 live smoke 后再 cutover |
| OI-004 | Control Plane | capacity / priority / cutover plan 还没有真实 mutation executor | 必须先绑定 approval、audit、allowlist、回滚和限流 |
| OI-005 | Media Assets | HTTP/HTTPS media cache recovery executor 已有，但平台私有源凭证、会话态重拉、自动后台重试和恢复后 AI enrichment 仍未统一 | Go 管 cache/audit/lifecycle，Python/provider worker 管私有源和 OCR/VLM/RAG |
| OI-006 | Knowledge / RAG | RAG dataset/index state 主要来自 checkpoint snapshot 推导 | 需要时接入外部 index metadata |
| OI-007 | Python AI Worker | worker 并发控制、autoscaling、优先级调度还没有真实执行器 | 基于 AgentJob pressure 和 approval 设计 executor |
| OI-008 | Frontend / Dashboard | 部分 runtime overview drilldown 仍依赖原始 JSON | 优先做只读表格化，不加入控制逻辑 |
| OI-009 | SDD Governance | 需要持续防止 TODO、BACKLOG、LIVE_CHECKS、OPEN_ISSUES 职责混用 | 每轮结束前同步检查四类文档 |

## 当前运行验证重点

这些不是代码 TODO，而是需要真实环境或人工操作验证：

- QQ 群 observe-only 是否稳定收集文本、图片、文件，且不主动回复。
- media asset content / recovery / retention 诊断是否能解释图片和附件无法显示的问题。
- QQ/NapCat Go adapter live send smoke 是否覆盖双 QQ 账号、私聊、群聊、图片、文件。
- AgentJob、knowledge worker、RAG ingest、image generation worker 是否通过 Go lifecycle 正常显示状态。
- NATS external lease、result-ack、queue topology 是否在 cutover 前通过 smoke。
- operator approval / control mutation audit 是否只记录审计，不产生未授权副作用。

完整清单见：

- `docs/sdd/LIVE_CHECKS.md`

## 下一步进入迭代的规则

1. 先看 `OPEN_ISSUES.md`，确认要解决的是问题、风险还是决策。
2. 如果本轮要做，把它拆成 3 到 10 条可完成任务写入 `TODO.md`。
3. 先写或更新 `specs/` 下的 SDD 设计。
4. 再改代码，保持 Go / Python 边界不漂移。
5. 跑相关测试和必要 smoke。
6. 更新 `DONE.md`、`OPEN_ISSUES.md`、`LIVE_CHECKS.md`、`BACKLOG.md`。
7. 本轮结束前清空 `TODO.md`，除非存在真实外部阻塞。

## 当前架构边界

Go 适合继续承接：

- 消息入口标准化、幂等、防循环、发送审计。
- Outbox / Inbox、AgentJob、MQ adapter、租约和重试。
- 媒体资产登记、访问控制、恢复计划、保留策略。
- 群观察状态、checkpoint、只读诊断和 dashboard control plane。
- operator approval、control mutation audit、只读 preflight 和未来受控 executor。

Python 应继续保留：

- LLM / MiMo / ChatGPT / OpenAI / Claude 等模型供应商适配。
- prompt、上下文组装、工具调用和 agent reasoning。
- OCR/VLM、图片生成、embedding、rerank、RAG 策略实验。
- 群 memory 抽取、攻略沉淀、知识蒸馏、RAG evaluation。

不要为了“更多 Go”把 AI 策略实验、模型调用和 provider-specific fallback 硬迁到 Go；Go 的价值是让 runtime 稳定、可审计、可回滚。
