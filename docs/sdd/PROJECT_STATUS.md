# Akashic 项目状态总览

最后更新：2026-06-04

本文件是项目状态总账，用来在重启 Codex、人工交接或开始新一轮迭代前快速判断“现在做到哪、还有什么坑、下一步从哪里进”。它不替代其它 SDD 文档，而是把它们的职责汇总到一个入口。

## 文档分工

| 文档 | 作用 | 使用规则 |
| --- | --- | --- |
| `TODO.md` | 本轮迭代必须全部完成的短待办 | 每轮结束前必须清空，或写明真实外部阻塞 |
| `DONE.md` | 已完成并通过检查的事实归档 | 只写已经落地的能力，不写计划 |
| `PROBLEM_REGISTRY.md` | 所有问题、风险、阻塞和待决策项的总入口 | 先看这里确认问题该进入哪个文档 |
| `OPEN_ISSUES.md` | 当前未解决问题、风险、待决策项详细表 | 新发现的问题先写这里，不要堆进 TODO |
| `BACKLOG.md` | 长期规划、候选任务、未来可做事项 | 进入迭代前再拆到 TODO |
| `LIVE_CHECKS.md` | 需要真实运行环境验证的检查项 | 不是开发任务，不要回填到 TODO |
| `specs/` | 每个能力的 SDD 设计契约 | 先写设计，再改代码 |
| `reviews/` | 每轮代码/架构审查记录 | 记录边界、测试和残余风险 |

## 当前迭代状态

当前 `TODO.md` 已清空。最近完成的切片包括 unified goal verifier 的
exact migration buckets、proactive runtime flow live smoke、dashboard
proactive tick-log runtime fallback live verifier、media asset HTTP/HTTPS
recovery executor live smoke、dashboard knowledge/RAG state boundary live
verifier、unified goal verifier 对 dashboard knowledge/RAG boundary 的
single-retry hardening、unified goal verifier 的 stdout-mode hardening，以及
agent-worker-status startup prune live smoke：

- `.\scripts\verify-go-migration-goal.ps1` 仍会默认回写
  `.codex-goal-verifier.json`，并在已有 `current_state` 之外新增顶层
  `migration_residuals`。
- `migration_residuals` 现已对以下八类残留给出稳定 machine-readable 分类、
  关键 live facts 和最小下一步：
  - `qq_napcat_cutover`
  - `telegram_backend`
  - `agent_job_external_lease_result_ack`
  - `scheduler_runtime`
  - `worker_control_executors`
  - `media_recovery_private_source_executor`
  - `dashboard_read_models`
  - `mq_adapter_boundary`
- 同时 unified artifact 现在还提供 `migration_bucket_summary`，把这些残留
  直接归入四个精确桶：
  - `already_in_go_only_missing_live_verification`
  - `go_control_plane_present_but_real_executor_missing`
  - `still_not_fully_cut_over`
  - `explicitly_python_owned`
- 当前 live artifact 已真实把以下状态固定下来：
  - QQ 仍是 `go_local_outbox_worker` + `account_conversation_kind_gated`
  - Telegram 仍是 `token_missing`
  - `agent_job external lease result-ack` 仍是 production `blocked`
  - scheduler 已是 `go_control_plane_mutation_and_recovery_live_verified`
  - worker control executors 仍是 `go_control_plane_live_verified_without_worker_control_executor`
  - dashboard fallback 仍是 `live_verified_runtime_read_models`
- 本轮还补了一条 verifier 稳定性收口：当
  `verify-dashboard-knowledge-rag-state-boundary.ps1` 首次采样未返回
  `live_verified` 时，`verify-go-migration-goal.ps1` 现在会做一次短暂重试，
  避免 `.codex-goal-verifier.json` 在同一 turn 上把已通过的 dashboard
  knowledge/RAG parity 误写成 `partially_live_verified_read_model`。
- 本轮还把 unified goal verifier 的 stdout 路径显式化：
  `verify-go-migration-goal.ps1` 现在支持 `-StdoutMode summary|full|none`。
  默认只输出小摘要 JSON，`none` 只刷新 `.codex-goal-verifier.json`，
  `full` 才输出完整 artifact。这样 current-turn 取证不再依赖交互式调用端
  吃完整大 JSON stdout。
- 同时，本轮还新增了 repo-owned `proactive` isolated live smoke：
  `.\scripts\verify-proactive-runtime-flow-live-smoke.ps1` 已真实证明 Go-owned
  proactive deliveries、seen/rejection/cleanup、anyaction quota、drift、
  bg-context 和 tick-log state transitions 在 temp runtime 中全部通过，且
  `.\scripts\verify-go-migration-goal.ps1` 已将其纳入
  `proactive_runtime_flow_smoke`。
- 本轮还修正了一个真实 runtime 假阻塞来源：Go runtime 现在会在
  repository load 时自动 prune stale `agent-worker-statuses`，避免历史
  Python heartbeat 残留在 restart 后重新进入内存。对应
  `.\scripts\verify-agent-worker-status-startup-prune-live-smoke.ps1`
  已真实证明 stale seed record 会在启动时被删掉而 active record 保留；当前
  8780 live runtime 重启后也已稳定回到
  `agent_workers_total=0/agent_workers_stale=0`，不再因为 stale residue
  把 `knowledge_job_planner_readiness` 和 agent-job capacity/priority 误报成
  stale-worker blocker。
- 此外，本轮还新增了 repo-owned dashboard fallback verifier：
  `.\scripts\verify-dashboard-proactive-tick-logs-boundary.ps1` 已真实证明在
  SQLite `tick_log` mirror 为空时，dashboard
  `/api/dashboard/proactive/tick_logs` 的 list/detail/steps 会回退到 Go
  runtime，而不会把 runtime tick 数据写回 SQLite；统一 verifier 也已把这条
  证据纳入 `dashboard_proactive_tick_logs_boundary`。
- 同时，本轮还新增了 repo-owned `media recovery` executor smoke：
  `.\scripts\verify-media-asset-content-recovery-live-smoke.ps1` 已真实证明
  Go 的 HTTP/HTTPS recovery executor 可以在 temp runtime 中完成
  approval-bound preflight、dry-run no-write、正式 recovery、registry update、
  content readback 和 applied control-mutation audit；统一 verifier 也已把这条
  证据纳入 `media_recovery_executor_smoke`，并把
  `media_recovery_private_source_executor` 残留分类升级为
  `go_http_https_executor_live_verified_private_source_executor_incomplete`。
- 本轮还补齐了 `Media Asset Content` 的 dashboard/runtime live parity：
  dashboard fallback 现在会读取 `/v1/media-assets/content-diagnostics`，
  保留 `media_asset_content_*` summary、`Media Asset Content` card/detail 和
  sampled asset 的 recovery/preflight endpoint；新脚本
  `.\scripts\verify-dashboard-media-asset-content-boundary.ps1` 已把这条边界固定为
  repo-owned live verifier，统一 artifact 也已纳入
  `dashboard_media_asset_content_boundary`。
- 本轮还补齐了 repo-local launcher 对 `agent_job external lease result-ack`
  preflight flags 的注入面：`scripts/start-agent-runtime.ps1` 现在支持显式传入
  runtime addr/state/shadow、queue backend/dsn/mode、external-lease cutover、
  dual-read/state-lease-workers-disabled、agent-job duplicate/flow smoke 和
  strict lease token，并且 `/v1/runtime-config` 已同步暴露这些布尔 cutover
  env 的 presence/redacted value。
- 同时，本轮新增了 repo-owned launcher live smoke：
  `.\scripts\verify-agent-job-external-lease-launcher-preflight.ps1` 已通过
  `start-agent-runtime.ps1 -Foreground` 在隔离 temp runtime 上真实证明：
  runtime-config / queue-backend / queue-topology / approval-bound preflight
  都已反映这组 result-ack flags。统一 goal verifier 也已把这条 evidence
  纳入 `agent_job_external_lease_launcher_preflight`，所以当前 production gap
  已进一步收敛到 live runtime flags / owner 尚未切换，而不是 launcher 无法
  reproducible 地带起 preflight 配置。
- 本轮又新增了 canonical launcher bundle 只读接口：
  `GET /v1/agent-job-external-lease/launcher-bundle`。它会把
  `start-agent-runtime.ps1` 所需的 exact launcher parameters、
  environment overrides、required external inputs、verification steps 和当前
  plan 一起固定成 machine-readable contract。对应
  `.\scripts\verify-agent-job-external-lease-launcher-bundle.ps1` 已在隔离
  temp runtime + temp NATS 上真实证明：
  - blocked runtime 下 bundle 会稳定返回 canonical NATS/result-ack flags、
    `QueueDSN` external input 和 verification steps；
  - 用 bundle 动态拼出来的 launcher 参数可以把 temp runtime 启到
    promoted state；
  - queue backend / queue topology / runtime-overview / approval-bound preflight
    会一起进入 `repo_owned_launcher_external_lease_bundle_live_verified`。
  这意味着当前剩余 gap 已进一步收敛到 production runtime 尚未注入这些 flags，
  而不是 launcher contract 缺失或 bundle 无法复跑。

对应设计：

- `docs/sdd/specs/agent-gateway/212-goal-verifier-migration-buckets.md`
- `docs/sdd/specs/agent-gateway/213-proactive-runtime-flow-live-smoke.md`
- `docs/sdd/specs/agent-gateway/214-dashboard-proactive-tick-logs-runtime-fallback-live-verifier.md`
- `docs/sdd/specs/agent-gateway/215-goal-verifier-dashboard-read-model-checks-alias.md`
- `docs/sdd/specs/agent-gateway/216-media-asset-content-recovery-http-executor-live-smoke.md`
- `docs/sdd/specs/agent-gateway/217-dashboard-knowledge-rag-state-boundary-live-verifier.md`
- `docs/sdd/specs/agent-gateway/220-dashboard-media-asset-content-boundary-live-verifier.md`

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

以下是 `PROBLEM_REGISTRY.md` / `OPEN_ISSUES.md` 的高层摘要，详细字段和下一步以 `OPEN_ISSUES.md` 为准。

| ID | 领域 | 未解决点 | 当前处理原则 |
| --- | --- | --- | --- |
| OI-001 | QQ / Telegram / Observe | QQ/NapCat Go adapter cutover 还没进入真实 live send 范围 | 先做 live smoke，再决定是否切换真实发送 |
| OI-002 | Queue / MQ | NATS JetStream 是当前推荐 MQ，Redis Streams / RabbitMQ adapter 尚未实现 | 只有出现明确部署需求才新增 adapter |
| OI-003 | AgentJob / Worker | `agent_job` external lease result-ack 仍在 readiness/plan/cutover 边界 | temp NATS smoke 与 isolated cutover preflight 已过，production flags/owner 到位后再 cutover |
| OI-004 | Control Plane | capacity / priority / cutover plan 还没有真实 mutation executor | 必须先绑定 approval、audit、allowlist、回滚和限流 |
| OI-005 | Media Assets | HTTP/HTTPS media cache recovery executor 已有且 repo-owned isolated live smoke 已过，但平台私有源凭证、会话态重拉、自动后台重试和恢复后 AI enrichment 仍未统一 | Go 管 cache/audit/lifecycle，Python/provider worker 管私有源和 OCR/VLM/RAG |
| OI-006 | Knowledge / RAG | RAG dataset/index state 主要来自 checkpoint snapshot 推导 | 需要时接入外部 index metadata |
| OI-007 | Python AI Worker | worker 并发控制、autoscaling、优先级调度还没有真实执行器 | 基于 AgentJob pressure 和 approval 设计 executor |
| OI-008 | Frontend / Dashboard | 当前 tracked runtime-overview drilldown 已基本结构化，后续只处理新增 read-model gap | 继续优先做只读表格化，不加入控制逻辑 |
| OI-009 | SDD Governance | 需要持续防止 TODO、BACKLOG、LIVE_CHECKS、OPEN_ISSUES 职责混用 | 每轮结束前同步检查四类文档 |

## 当前运行验证重点

这些不是代码 TODO，而是需要真实环境或人工操作验证：

- QQ 群 observe-only 是否稳定收集文本、图片、文件，且不主动回复。
- media asset content / recovery / retention 诊断是否能解释图片和附件无法显示的问题。
- `media_asset_content_recovery` 的 runtime/dashboard parity 现在已 live verified；media 线剩余关注点已收敛到 private-source fetch、后台重拉与恢复后 AI enrichment，而不是 dashboard 读模型。
- QQ/NapCat Go adapter live send smoke 是否覆盖双 QQ 账号、私聊、群聊、图片、文件。
- AgentJob、knowledge worker、RAG ingest、image generation worker 是否通过 Go lifecycle 正常显示状态。
- NATS external lease、result-ack、queue topology 是否在 cutover 前通过 smoke。
- operator approval / control mutation audit 的 ledger / preflight / policy /
  runtime-overview 一致性现在已 live verified；剩余要观察的是未来真实
  executor 落地后是否仍不产生未授权副作用。
- worker-status stale residue 当前已具备显式 cleanup mutation 和 repo-owned
  live smoke；剩余观察点不再是 stale heartbeat 清理，而是未来真实 executor
  落地后是否仍保持 worker 数与 runtime-overview summary 一致。

完整清单见：

- `docs/sdd/LIVE_CHECKS.md`
- `docs/sdd/REMAINING_GO_MIGRATION.md`

## 下一步进入迭代的规则

1. 先看 `PROBLEM_REGISTRY.md` 和 `OPEN_ISSUES.md`，确认要解决的是问题、风险还是决策。
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

补充：receiver-status stale cleanup 现在也已有明确 Go mutation 与 repo-owned
live smoke；当前缺口不再是“只能看到 heartbeat_stale 但没有受控删除路径”，而是
是否要在长期运行 runtime 上主动应用 cleanup。

补充：dashboard knowledge/RAG parity 现在也已补齐；`/api/dashboard/runtime-overview`
不再只返回 `knowledge_pipelines` card/detail 而遗漏 top-level payload。

补充（2026-06-04）：

- 本轮新增 `GET /v1/agent-job-external-lease/cutover-diff`，把当前 runtime 与
  canonical launcher bundle 的 drift 直接固定成 machine-readable 结果。
- `scripts/verify-agent-job-external-lease-cutover-diff.ps1` 已真实证明：
  当前 8780 live runtime 仍明确 blocked，同时同一 bundle 带起的 temp
  NATS/runtime 会先表现为 `zero config drift + worker coverage blocked`，
  再在 knowledge worker heartbeat 后进入 ready。
- 为加载新 handler，本轮受控重启了 8780 上的 `agent-runtime`，并恢复到既有
  QQ live gate；当前 fresh artifact 已再次确认
  `outbox_execution_owner=go_local_outbox_worker`、
  `outbox_execution_scope=account_conversation_kind_gated`、
  `agent_job_execution_owner=python_ai_worker_state_store_lease`、
  `agent_job_ack_owner=go_state_store_api`。
