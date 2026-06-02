# Akashic Problem Registry

最后更新：2026-06-02

本文件是 Akashic Agent Go 化改造的“问题登记册”入口，用来统一说明所有问题、风险、待决策项和阻塞项应该放在哪里、如何进入迭代、如何关闭。

`OPEN_ISSUES.md` 仍是当前未解决问题的详细表；本文件负责提供总入口和治理规则，避免 TODO、DONE、BACKLOG、LIVE_CHECKS、OPEN_ISSUES 之间职责混用。

## 文档分工

| 文档 | 是否问题入口 | 内容 | 关闭规则 |
| --- | --- | --- | --- |
| `PROBLEM_REGISTRY.md` | 是 | 所有问题文档的总入口、分类规则和当前 open issue 索引 | 不直接关闭具体问题，关闭动作落到 `OPEN_ISSUES.md` / `DONE.md` |
| `OPEN_ISSUES.md` | 是 | 当前尚未解决的问题、风险、阻塞、待决策项和后续缺口 | 解决后从 open 表移除或改为 resolved 记录，并同步 `DONE.md` |
| `TODO.md` | 否 | 本轮必须全部完成的短任务 | 本轮结束前清空，只有真实外部阻塞可留下 |
| `DONE.md` | 否 | 已落地、已验证的事实 | 只追加完成事实，不承载未来问题 |
| `BACKLOG.md` | 否 | 未来候选任务、长期规划、非本轮事项 | 进入迭代前拆成 `TODO.md` 项 |
| `LIVE_CHECKS.md` | 否 | 需要真实环境、人工 smoke 或线上观察的检查项 | 验证完成后同步结果到 `DONE.md` 或 `OPEN_ISSUES.md` |

## 问题分类

| 类型 | 写入位置 | 示例 |
| --- | --- | --- |
| 当前未解决问题 | `OPEN_ISSUES.md` | RAG 外部索引状态还不能被 Go 直接证明 |
| 当前迭代必须解决的问题 | 先写 `OPEN_ISSUES.md`，再拆到 `TODO.md` | 本轮要实现某个 executor 前发现缺少审批审计 |
| 真实环境验证问题 | `LIVE_CHECKS.md`，失败后再写 `OPEN_ISSUES.md` | QQ/NapCat live send smoke 未覆盖群图片 |
| 未来可做但非问题 | `BACKLOG.md` | 新增 RabbitMQ adapter 的候选计划 |
| 已解决问题 | `DONE.md`，必要时在 review 中记录证据 | media content recovery executor 已完成并通过测试 |

## 当前未解决问题索引

当前所有 open issue 的权威字段以 `OPEN_ISSUES.md` 为准。本索引只提供进入问题总账时的快速目录。

| ID | 领域 | 摘要 | 权威记录 |
| --- | --- | --- | --- |
| OI-001 | QQ / Telegram / Observe | QQ/NapCat Go adapter cutover 尚未进入真实 live send 范围 | `OPEN_ISSUES.md` |
| OI-002 | Queue / MQ | Redis Streams / RabbitMQ adapter 尚未实现 | `OPEN_ISSUES.md` |
| OI-003 | AgentJob / Worker | external lease result-ack 仍处于 readiness/plan/smoke 边界 | `OPEN_ISSUES.md` |
| OI-004 | Control Plane | capacity / priority / cutover plan 还没有真实 mutation executor | `OPEN_ISSUES.md` |
| OI-005 | Media Assets | 平台私有源凭证、会话态重拉、自动后台重试和恢复后 AI enrichment 尚未统一 | `OPEN_ISSUES.md` |
| OI-006 | Knowledge / RAG | RAG dataset/index state 主要来自 checkpoint snapshot 推导 | `OPEN_ISSUES.md` |
| OI-007 | Python AI Worker | worker 并发控制、autoscaling、优先级调度还没有真实控制面执行器 | `OPEN_ISSUES.md` |
| OI-008 | Frontend / Dashboard | 部分 runtime overview drilldown 仍依赖原始 JSON | `OPEN_ISSUES.md` |
| OI-009 | SDD Governance | 需要持续防止 TODO、BACKLOG、LIVE_CHECKS、OPEN_ISSUES 职责混用 | `OPEN_ISSUES.md` |

## 进入迭代规则

1. 新问题先进入 `OPEN_ISSUES.md`，除非它只是未来想法或 live smoke 项。
2. 如果本轮要解决某个 open issue，先把它拆成 3 到 10 条可完成任务写入 `TODO.md`。
3. `TODO.md` 中的任务必须在本轮完成；不能把长期问题直接堆进 TODO。
4. 完成后更新 `DONE.md`、相关 spec/review，并从 `OPEN_ISSUES.md` 移除或改写该问题状态。
5. 如果验证只能在真实环境完成，开发完成事实写入 `DONE.md`，剩余 live 验证写入 `LIVE_CHECKS.md`。

