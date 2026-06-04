# Akashic Runtime Goal Prompt

Use this prompt when resuming the Akashic Go migration in a new Codex thread.

```text
继续 Akashic runtime Go migration。

工作目录：E:\agent\my-akashic_agent

目标：
- 不重复做已经 live verified 的 read-model 或 control-plane 收尾。
- 只推进一个真实、可闭环、可验证的剩余切片。
- 每轮必须把当前 TODO 清零，或者把未完成项明确归档为外部 blocker。

先做 current-turn 取证：
- git status --short
- powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary
- 读取：
  - docs/sdd/TODO.md
  - docs/sdd/PROJECT_STATUS.md
  - docs/sdd/DONE.md
  - docs/sdd/OPEN_ISSUES.md
  - docs/sdd/LIVE_CHECKS.md
  - docs/sdd/REMAINING_GO_MIGRATION.md
  - docs/sdd/PROBLEM_REGISTRY.md
  - docs/sdd/BACKLOG.md
  - docs/atdd/README.md
  - docs/tdd/README.md
  - 本轮相关 docs/sdd/specs/agent-gateway/*.md 与 docs/sdd/reviews/*.md

TODO / 文档规则：
- docs/sdd/TODO.md 是本轮契约，不是长期愿望清单。
- 本轮开始前先把要做的 3-10 条短任务写进 TODO.md。
- 本轮结束前必须清空 TODO.md；不能把“下一轮继续同一 TODO”当成交付。
- 新发现的问题先写 OPEN_ISSUES.md，不要堆回 TODO.md。
- 未来候选项写 BACKLOG.md。
- 需要长期 live 观察的写 LIVE_CHECKS.md。
- 每个真实切片都要补：
  - SDD spec
  - ATDD
  - TDD
  - review
  - 必要测试 / smoke
  - DONE.md / PROJECT_STATUS.md / REMAINING_GO_MIGRATION.md / 相关索引

当前 canonical artifact：
- repo 根有 .codex-goal-verifier.json
- verify-go-migration-goal.ps1 支持 -StdoutMode summary|full|none
- 默认以 current_state、migration_residuals、migration_bucket_summary 为当前 turn 权威摘要
- 如果引用 live 状态，先重新跑 verifier，不要复述旧 handoff

当前 live facts（进入实现前先复核；以下是当前默认预期）：
- goal_ready_to_close=false
- open_blockers=["telegram_token_missing","qq_image_native_platform_blocker_unresolved"]
- qq_group_send_enabled=false
- telegram_token_configured=false
- outbox_execution_owner=go_local_outbox_worker
- outbox_execution_scope=account_conversation_kind_gated
- agent_job_execution_owner=python_ai_worker_state_store_lease
- agent_job_ack_owner=go_state_store_api
- agent_job_external_lease_ready=false
- agent_job_external_lease_decision=blocked
- receiver_statuses_telegram=0
- agent_workers_total=0
- agent_workers_stale=0
- runtime_overview_agent_workers_stale=0
- dashboard_fallback_category=live_verified_runtime_read_models

当前 migration buckets（进入实现前先复核）：
- already_in_go_only_missing_live_verification
  - scheduler_runtime
  - knowledge_rag_state_boundary
  - dashboard_read_models
  - mq_adapter_boundary
- go_control_plane_present_but_real_executor_missing
  - worker_control_executors
  - media_recovery_private_source_executor
- still_not_fully_cut_over
  - qq_napcat_cutover
  - telegram_backend
  - agent_job_external_lease_result_ack
- explicitly_python_owned
  - python_owned_surfaces

当前优先级：
1. production agent_job external lease result-ack flags / owner preflight
2. Telegram backend getMe + receiver + send/receive smoke（前提：TELEGRAM_BOT_TOKEN 已到位）
3. QQ rich-media / native platform blocker 继续定位（前提：明确要继续碰平台会话）

执行规则：
- 不要重复收已经 live verified 的 dashboard/read-model 细枝末节。
- 不要把 Python-owned AI runtime 当成 Go 迁移未完成项。
- 如果做 runtime / dashboard / verifier 改动，必须重跑对应 live verifier。
- 如果改 Go handler / service / store，先补或更新 Go test，再跑 smoke。
- 如果改 dashboard fallback，必要时重启 main.py 或相关本地进程，让 live verifier 吃到新代码。
- 不要为了“更多 Go”把 prompt、LLM/VLM/OCR、RAG ranking、image generation 迁入 Go。

Go / Python 边界：
- Go 负责：
  - inbox/outbox
  - idempotency / anti-loop
  - queue / lease / retry / checkpoint
  - scheduler control plane
  - media asset lifecycle / retention / controlled recovery
  - runtime diagnostics / runtime overview / dashboard read-model
  - operator approval / control mutation audit / deterministic state
- Python 负责：
  - LLM/provider 调用
  - prompt/context/reasoning
  - tool execution
  - OCR / VLM
  - image generation
  - memory / RAG 算法与策略实验
  - provider-specific private-source enrichment

交付口径：
- 代码、测试、live verifier、SDD 文档、状态文档一起闭环。
- 最终答复必须直接给出：
  - 本轮做了什么
  - 实际跑了哪些验证
  - 当前 blocker 是否变化
  - goal 是否仍 active

当前最小 blocker：
1. telegram_token_missing
2. qq_image_native_platform_blocker_unresolved
```
