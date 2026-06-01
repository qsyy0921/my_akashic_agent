# Akashic Go 迁移完成记录

最后更新：2026-06-01

本文件只做阶段归档，完整逐切片审查记录见 `docs/sdd/reviews/README.md` 和各 phase review 文档。

## 架构与服务边界

- Go 服务边界已统一为 `services/agent-runtime`，采用 DDD + 六边形架构分层。
- Go 负责 agent runtime/control plane，Python 负责模型调用、prompt、Memory/RAG/chunking/retrieval、OCR/VLM、工具执行、群知识沉淀和快速实验。
- Go runtime 默认使用 `.akashic-workspace/agent-runtime` 文件态保存确定性状态，可用 `AKASHIC_RUNTIME_STATE_DIR` 或单项 DSN/PATH 覆盖。
- SDD 任务文档已拆分为 `TODO.md`、`DONE.md`、`LIVE_CHECKS.md`、`OPEN_ISSUES.md`、`BACKLOG.md`：TODO 只保存本轮必须完成项，OPEN_ISSUES 记录所有未解决问题，避免长期规划或风险缺口导致 TODO 膨胀。
- `OPEN_ISSUES.md` 已纳入 SDD 入口说明、迭代流程和 review gate：每轮开始必须读取，留下风险、缺口、阻塞或待决策项时必须同步更新。

## 消息、资产与发送链路

- 已实现 Go shadow audit、inbox、media asset registry、安全 content 访问和 dashboard 附件代理。
- 已实现 Go send ledger、private echo 判断和双账号防循环相关只读诊断。
- 已实现 Go outbox 生命周期、outbox event stream、delivery dispatch plan、readiness、smoke readiness 和 metrics。
- 已实现 Go outbox account pressure 只读诊断：按 `channel_kind:account_id` 汇总 queued/dispatching/active/dead-letter，并接入 runtime overview，为后续账号级限流提供数据基础。
- 已实现 Go local outbox delivery worker 账号级发送节流：支持 min interval / window limit，lease 前跳过被节流账号，blocked delivery 保持 queued，不误标 failed。
- 已实现 Go external lease outbox executor 账号级发送节流：复用同一 account rate limiter，在租约前判断被节流账号并返回 `nack/delivery_rate_limited`，不拿 outbox lease、不调用 delivery adapter，避免未来 NATS cutover 绕过限流。
- 已实现 Telegram DeliveryAdapter、QQ/NapCat OneBot HTTP/WebSocket DeliveryAdapter、adapter diagnostics、live health 和 runtime config 脱敏诊断。
- 已加入可选 Go local outbox delivery worker，默认关闭，避免未完成 cutover 时触发真实平台发送。
- 已实现 outbound cutover readiness：`/v1/outbound-cutover/readiness` 聚合 OneBot 配置、delivery smoke matrix、queue backend execution owner 和 runtime worker 状态，判断 QQ/NapCat 发送链路是否可交给 Go 执行，且不发送平台消息。
- 已实现 outbound cutover plan：`/v1/outbound-cutover/plan` 在 readiness 基础上输出 Go local outbox worker 或 NATS external lease 的启用步骤、验证入口和回滚步骤，保持只读且不改环境变量、不发平台消息。
- runtime overview 已聚合 outbound cutover plan：summary/card/detail 可直接看到当前/目标/推荐 execution owner、决策和 blocker 数，避免 dashboard/operator 入口漏看 QQ/NapCat cutover 门禁。
- runtime overview 已聚合 delivery smoke readiness：`Delivery Smoke` card 和 `delivery_smoke_*` summary 可直接展示默认 QQ/Telegram smoke matrix 是否具备 adapter 路由条件，仍不发送平台消息、不租约 outbox、不触发 Python AI。
- 已实现 `agent_job` external lease result-ack readiness：`/v1/agent-job-external-lease/readiness` 聚合 queue backend gate、strict lease token、runtime flag、AgentJob pressure 和 Python worker coverage，判断是否可以把 generic job 队列确认权扩展到 NATS external lease；Go 仍不执行 AI job。
- 已实现 `agent_job` external lease result-ack plan：`/v1/agent-job-external-lease/plan` 输出只读启用、验证和回滚步骤，明确 Python AI worker 继续执行模型/RAG/memory/OCR/VLM/图片任务，Go 只规划确定性 AgentJob 生命周期确认权。
- runtime overview 已聚合 `agent_job` external lease result-ack plan：summary/card/detail 可直接看到当前/目标/推荐 execution owner、决策、blocker 数和回滚步骤，仍保持只读且不 ack/nack MQ、不执行 AI job。
- Python dashboard 已规范化 Go-owned runtime control-plane detail：`agent_job_capacity_plan`、`agent_job_external_lease_readiness`、`agent_job_external_lease_plan`、`outbound_cutover_plan` 现在都可通过 `/api/dashboard/runtime-overview` 以稳定 summary/card/detail 读取，仍不接管 worker、MQ ack/nack、outbox 发送或 AI job 执行。

## 观察群与接收链路

- 已实现 Go observe target sync/list、receiver status、receiver lease、observe capture diagnostics。
- 已实现 Go-owned receiver lease cleanup：`POST /v1/receiver-leases/cleanup-expired` 可显式删除过期接收端租约，返回 deleted/remaining summary，并保持 `side_effect=runtime_state_only`；QQ/Telegram receiver 进程、observe-only 群采集和 Python AI pipeline 不受影响。
- runtime overview 和 Python dashboard fallback 已暴露 receiver lease cleanup 可见性：`receiver_lease_cleanup_required`、`receiver_lease_cleanup_endpoint` 和 `Receiver Leases` active/expired card value 可直接提示过期租约是否需要人工清理，但不会自动调用 cleanup 或影响接收端。
- 已实现 Go inbox metrics、receiver stale 降级、recent inbox activity 推断 receiver connected。
- 已实现 Telegram 和 QQ/NapCat inbound dedupe，包含 observe-only 群文本、图片、普通群消息、私聊和群文件上传 notice。
- 已实现 inbound dedupe metrics，并接入 runtime overview/dashboard。
- 已实现 Go-owned media asset content diagnostics：`/v1/media-assets/content-diagnostics` 可按 asset/filter 输出 ready/forbidden/unavailable/disabled/error，帮助前端定位图片/文件无法显示原因；Go 只做安全根目录访问预检，Python 继续负责 OCR/VLM/文件理解。
- 已实现 Go-owned media asset retention diagnostics：`/v1/media-assets/retention-diagnostics` 可按 retention policy、age 和 TTL dry-run 参数输出 cleanup due advisory，明确 `side_effect=none`，不删除 registry、不删除文件、不执行 OCR/VLM/文件解析。
- runtime overview 已聚合 `Media Asset Content`：summary/card/detail 直接展示最近附件内容 ready/forbidden/unavailable/disabled/error 状态，方便 dashboard 定位 QQ 图片/文件显示问题，仍不执行 OCR/VLM/文件解析。
- runtime overview 和 Python dashboard fallback 已聚合 `Media Asset Retention`：summary/card/detail 可直接看到 cleanup_due、permanent/default/ephemeral/unknown 统计和只读 retention diagnostics，仍不删除 registry 或文件，不触发 OCR/VLM/文件解析。
- 已实现 Go-owned media asset retention cleanup plan：`/v1/media-assets/retention-plan` 基于 retention diagnostics 输出只读 dry-run 清理计划、候选资产、operator approval/control mutation audit 绑定步骤、验证/回滚步骤和 `side_effect=none`；当前不删除 media metadata 或本地文件内容。
- runtime overview 和 Python dashboard fallback 已聚合 `Media Asset Retention Plan`：summary/card/detail 可直接看到 retention cleanup plan 是否 ready、候选数、blocker 数和 required steps；Python 只做只读展示，不创建 approval/mutation、不删除文件、不触发 OCR/VLM/AI。
- Go-owned control mutation policy 已支持 `media_asset_retention / cleanup_expired`：retention plan 输出的审批和 planned mutation audit 链路可通过 preflight 识别，unsupported media retention action 仍被阻断；当前仍不实现真实清理执行器、不删除 metadata 或文件。

## AgentJob、队列与 worker 状态

- 已实现 Go generic AgentJob 生命周期、文件态持久化、dedupe_key、event stream、metrics、dashboard 恢复入口。
- 已实现 AgentJob lease token、heartbeat/renew、strict token 模式、recover-expired、queue work id 精确租约入口。
- 已完成 NATS JetStream shadow_publish、dual_read_compare、external_lease 诊断门禁和 outbox external lease smoke。
- external_lease cutover gate 已增加 local outbox worker 冲突检查：`AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 时 gate 保持 blocked，防止 state-store worker 与 NATS external lease 同时执行真实发送。
- 已实现 agent_job external lease result-ack 映射和 subject 扩容门禁，当前仍需显式 smoke/cutover flag。
- `/v1/queue-backend` 已增加 MQ provider capability matrix：明确 NATS JetStream 是当前推荐第一外部 MQ，暴露 shadow/dual-read/external-lease/agent_job result-ack、多 goroutine consumer 和 delayed nack 能力，并把 Redis Streams / RabbitMQ 标记为后续可替换 adapter 边界。
- `/v1/queue-backend` 已增加 execution owner 诊断：outbox delivery 可明确区分 `go_state_store_api`、`go_local_outbox_worker`、`nats_external_lease`；agent_job 可明确区分 Python 通过 state-store lease 执行，或 Python 执行后经 NATS result-ack 回确认。
- 已实现 Go-owned queue topology read model：`/v1/queue-topology` 将 queue backend、provider capability、execution owner 和 external lease gate 派生成 nodes/edges/work_kinds，直接展示 `outbox_delivery` 与 `agent_job` 的 queue source、execution owner、ack owner 和 blockers；全程只读，不 publish/lease/ack/nack/term MQ，不执行 AI job。
- runtime overview 已聚合 queue topology：summary/card/detail 现在可直接展示 queue topology 的 nodes/edges/work_kinds、external lease readiness、outbox execution owner、agent_job execution owner 和 ack owner；Python dashboard 也已规范化 `queue_topology` detail，仍保持只读且不执行 MQ/outbox/AgentJob/AI。
- runtime overview dashboard 的 `Queue Topology` detail 已从纯 JSON 提升为只读表格：展示 provider/mode/phase/external lease、nodes/edges 计数，以及每个 work kind 的 queue source、execution owner、ack owner、allowed 状态和 blockers；渲染时不 publish/lease/ack/nack/term MQ，不创建或执行 AgentJob/outbox，不触发 Python AI。
- external_lease 已增加 Go-owned 执行诊断：`/v1/queue-backend` 可查看 ack/nack/term disposition、reason、work_kind 统计和 bounded recent executions；只读诊断不执行 Python AI job。
- runtime overview 已聚合 external_lease 执行诊断：summary/card 可直接查看执行总数、错误数、ack/nack/term 分布，原始 queue backend detail 仍保留。
- runtime overview summary 已聚合 queue execution owner 字段：`queue_outbox_execution_owner` 和 `queue_agent_job_execution_owner` 能直接说明当前执行边界，避免把 NATS result-ack 误解为 Go 执行 AI job。
- 已实现 Go-owned `AgentJob` pressure 诊断：`/v1/job-metrics` 可按 `job_type` 查看 pending / leased / running / active / oldest_pending_age，并标记 high pressure；runtime overview 也聚合了 `agent_job_pressure_*` summary 与 `Agent Job Pressure` card。
- runtime overview 已新增 `Agent Job Worker Coverage`：把 Go-owned `AgentJob` pressure 与 Python worker heartbeat / stale / failed 状态关联起来，直接解释 backlog 是否由无 active worker、stale worker 或 failed worker 导致，不改变 Python 执行边界。
- 已实现 Go-owned `AgentJob Capacity Plan`：`/v1/agent-job-capacity/plan` 将 job pressure 与 Python worker coverage 转为只读运维建议，可区分需要恢复 worker、检查 failed/stale worker、或在已有 active worker 下调优并发/优先级；Go 不启动 worker、不执行 AI job、不 ack/nack MQ。
- 已实现 Go-owned `AgentJob Priority Plan`：`/v1/agent-job-priority/plan` 将 job pressure 与 Python worker coverage 转为确定性优先级/背压建议，输出 rank、priority class、score、action 和 blockers；Go 仍不做真实调度、不启动 worker、不改 MQ、不执行 AI job。
- runtime overview 已聚合 `Agent Job Capacity`：summary/card/detail 可直接看到 capacity ready/reason/blockers、blocked/warning/high-pressure job types、max pending/active 和完整 plan，仍保持只读且不启动 Python worker、不调度、不执行 AI。
- runtime overview 已聚合 `Agent Job Priority`，Python dashboard 也已规范化该 detail：summary/card/detail 可直接看到 priority ready/reason/blockers、high/blocked/warning job types 和 max priority score，仍保持只读且不改变调度、worker、MQ 或 AI 执行。
- 已实现 Go-owned `Operator Approval Ledger`：`POST /v1/operator-approvals` 和 `GET /v1/operator-approvals` 可记录/查询 operator 对控制面计划的 approved/rejected/revoked 审计，默认持久化到 `.akashic-workspace/agent-runtime/operator-approvals.json`，并明确 `side_effect=runtime_state_only`，不修改配置、不执行 cutover、不触发 Python AI。
- 已实现 Go-owned `Operator Approval Check`：`GET /v1/operator-approvals/check` 可按 `approval_id` 或 `target_kind + target_id` 校验 active approved 记录，返回 `approved/reason/blockers/approval/side_effect=none`，为后续真实控制面 mutation 绑定 approval id 提供只读 preflight。
- 已实现 Go-owned `Control Mutation Audit Ledger`：`POST /v1/control-mutations` 和 `GET /v1/control-mutations` 可记录/查询 planned/applied/failed/rolled_back 控制面变更审计，默认持久化到 `.akashic-workspace/agent-runtime/control-mutations.json`，并明确 `side_effect=runtime_state_only`，不修改配置、不执行 cutover、不启动 worker、不 ack/nack MQ、不触发 Python AI。
- 已实现 Go-owned `Control Mutation Preflight`：`GET /v1/control-mutations/preflight` 可在真实控制面变更前校验 active approved approval，返回 blockers 和建议 planned mutation audit 绑定信息，明确 `side_effect=none`，不创建审计、不修改配置、不执行 cutover、不启动 worker、不 ack/nack MQ、不触发 Python AI。
- 已实现 Go-owned `Control Mutation Policy`：preflight 现在会在 approval 校验前先按 domain policy 校验支持的 target/action，unsupported target/action 返回稳定 blocker，防止后续控制面执行器误把任意字符串变成可执行 mutation。
- 已实现 Go-owned `Control Mutation Policy API`：`GET /v1/control-mutations/policy` 可只读查询当前支持的控制面 mutation target/action，支持按 `target_kind` 过滤，返回 `side_effect=none`，避免 Python/dashboard 复制 allowlist。
- runtime overview 已聚合 Go-owned control mutation policy：summary/card/detail 可直接看到 allowlist target/action 数量与只读 policy 明细，Python/dashboard 不需要维护控制面 mutation allowlist。
- Python dashboard 已规范化 Go-owned control mutation policy：`/api/dashboard/runtime-overview` 稳定透传 `control_mutation_policy` detail、summary 默认值和 card；Python 仍只做展示适配，不维护 allowlist、不执行控制面变更。
- runtime overview dashboard 的 `Control Mutation Policy` detail 已从纯 JSON 提升为只读表格：展示 allowed/reason/target/action totals，以及 target kind 到 allowed actions 的 allowlist；渲染时不编辑 policy、不创建 approval/mutation、不执行 cutover、worker scaling、MQ ack/nack、media cleanup 或 AI。
- runtime overview 已聚合 Go-owned control audit：summary/card/detail 可直接看到 operator approvals、active approvals、control mutations、failed/rolled_back mutation audits 和 bounded recent records，仍保持只读且不记录 approval/mutation、不执行任何控制面变更。
- runtime overview dashboard 的 `Control Audit` detail 已从纯 JSON 提升为只读表格：展示 operator approval totals、control mutation totals、recent approval rows 和 recent mutation rows，并保留 raw JSON fallback；渲染时不创建/检查/撤销 approval，不记录 mutation，不执行 cutover、worker scaling、MQ ack/nack、media cleanup 或 AI。
- 已实现 Go-owned media asset retention cleanup preflight：`GET /v1/media-assets/retention-cleanup/preflight` 聚合 retention plan 与 control mutation preflight，校验 active approval 并返回建议 planned audit，明确 `side_effect=none`，不删除媒体 metadata/文件、不创建 approval/mutation、不触发 OCR/VLM/RAG/AI。
- 已实现 Go-owned media asset retention metadata cleanup executor：`POST /v1/media-assets/retention-cleanup` 通过 active approval preflight 后仅删除 Go 媒体登记元数据，并记录 applied/failed control mutation audit；dry-run 和 preflight 失败不产生副作用，且不删除本地文件、不触发 OCR/VLM/RAG/AI。
- runtime overview 已聚合 media retention cleanup：summary/card/detail 可直接看到候选数、recent cleanup audits、applied/failed 计数和 plan/preflight/cleanup endpoint，仍保持只读且不创建 approval/mutation、不执行 cleanup、不删除 metadata/文件、不触发 Python/OCR/VLM/RAG/AI。
- Python dashboard 已规范化 Go-owned media retention cleanup：`/api/dashboard/runtime-overview` 稳定透传 `media_asset_retention_cleanup` summary/card/detail 和 fallback 默认值，仍保持只读且不创建 approval/mutation、不执行 cleanup、不删除 metadata/文件、不触发 Python/OCR/VLM/RAG/AI。
- 已实现 Go-owned media asset content access plan：`GET /v1/media-assets/content-access-plan?asset_id=...` 可在前端点击附件前返回 ready/reason/blockers/content endpoint/verify/fallback steps，保持 `side_effect=none`，不流式返回内容、不下载远程媒体、不触发 OCR/VLM/RAG/AI。
- Python dashboard media content proxy 已接入 Go-owned access plan fallback：`/api/dashboard/media-assets/content` 在 Go content route 失败且旧 workspace fallback 不可用时返回结构化 `content_access_plan`，Python 不复制访问策略、不下载远程媒体、不触发 OCR/VLM/RAG/AI。
- Python dashboard 已新增 Go-owned media content access plan 只读代理：`/api/dashboard/media-assets/content-access-plan?asset_id=...` 直接透传 Go plan detail，前端可在打开附件前查看 ready/reason/blockers，Python 不复制访问策略、不触发 OCR/VLM/RAG/AI。
- dashboard 消息列表/详情的 `media_assets` 已新增 `content_access_plan_url`：前端可从每条消息的媒体资产直接跳到 Go-owned access plan 代理，Python 不主动批量拉取 plan、不复制访问策略、不触发 OCR/VLM/RAG/AI。
- 已新增 `MediaAssetContentAccessPlan` 合同 fixture：固化 Go `/v1/media-assets/content-access-plan` 与 dashboard `/api/dashboard/media-assets/content-access-plan` 的字段边界，包括 ready/reason/blockers/content endpoint/dashboard path/side_effect。
- Go contract fixture tests 已覆盖 `MediaAssetContentAccessPlan`：Go 侧现在会加载 `media_asset_content_access_plan.qq.image.json`，并校验 ready/reason/path/content endpoint/side_effect/required steps，防止 Go/Python access plan 字段漂移。
- Go `MediaAssetContentAccessPlan` response 已补齐 `runtime_path`、`dashboard_path`、`content_url`：前端和 dashboard 可直接消费 Go 生成的确定性链接，Python 仍只做代理/展示，不复制媒体访问策略或触发 OCR/VLM/RAG/AI。
- 已实现 Go-owned media content recovery plan：`GET /v1/media-assets/content-recovery-plan?asset_id=...` 基于 access plan 输出 disabled/forbidden/unavailable/error 的只读恢复步骤、future executor scope 和 URL hints，保持 `side_effect=none`，不下载/恢复/流式传输内容、不触发 OCR/VLM/RAG/AI。
- Python dashboard 已新增 media content recovery plan 只读代理：`/api/dashboard/media-assets/content-recovery-plan?asset_id=...` 透传 Go plan detail，浏览器可直接查看恢复建议，Python 不复制访问策略、不下载远程媒体、不触发 OCR/VLM/RAG/AI。
- Go media content diagnostics item 已新增 `content_recovery_plan_endpoint`，dashboard 消息列表/详情的 `media_assets` 已新增 `content_recovery_plan_url`，前端可从每条附件直接进入 Go-owned recovery plan；列表/详情渲染不主动请求 plan、不下载媒体、不触发 OCR/VLM/RAG/AI。
- Go media content diagnostics item 已新增 `content_access_plan_endpoint`，Python runtime overview dashboard 会规范化并在旧 runtime 未返回时按 asset id fallback 生成，使批量诊断可直接跳到单资产 access plan。
- 已实现 Go-owned media content recovery preflight：`GET /v1/media-assets/content-recovery/preflight` 将现有 recovery plan 绑定到 `media_asset_content/recover_content` control mutation policy、operator approval check 和 suggested planned audit；调用后不创建 approval/mutation、不下载/恢复/缓存内容、不流式传输内容、不触发 OCR/VLM/RAG/AI。
- Go-owned control mutation policy 已支持 `media_asset_content / recover_content`，为未来受控 media downloader/cache executor 提供 allowlist 门禁；当前仍不实现真实下载、恢复或缓存执行器。
- Python dashboard 已代理 Go-owned media content recovery preflight：`/api/dashboard/media-assets/content-recovery/preflight` 透传 `asset_id/target_id/operator_id/approval_id` 到 Go，并在 message `media_assets` 中新增 `content_recovery_preflight_url`；列表/详情渲染只暴露链接，不主动请求 preflight，不创建 approval/mutation，不下载/恢复/缓存内容，不触发 OCR/VLM/RAG/AI。
- Go media content diagnostics item 已新增 `content_recovery_preflight_endpoint`，runtime overview dashboard 的 `Media Asset Content` 表格也新增 preflight 链接；打开 detail 只展示 Go-owned endpoint，不主动调用 preflight，不创建 approval/mutation，不下载/恢复/缓存内容，不触发 OCR/VLM/RAG/AI。
- runtime overview dashboard 的 `Media Asset Content` detail 已从纯 JSON 提升为只读表格：展示 assets/ready/forbidden/unavailable/disabled/error totals、asset/name/status/reason，并提供 content/access/recovery 三类链接；渲染时不额外请求 plan、不打开内容、不下载媒体、不触发 OCR/VLM/RAG/AI。
- `services/agent-runtime/README.md` 和 SDD index 已同步当前 Go media asset API：content diagnostics/access plan、retention diagnostics/plan、cleanup preflight/executor 及各自 side-effect 边界。
- runtime overview 已聚合 `agent_job` external lease readiness：summary/card/detail 直接展示 result-ack gate、strict token、execution owner/scope 和 Python worker coverage blockers，仍保持只读且不执行 AI job。
- 已实现 Go-owned `Knowledge Pipeline Diagnostics`：按 observe-only QQ 群聚合 capture、`group_memory_extract` / `rag_ingest`、checkpoint 和 worker coverage，并接入 runtime overview `Knowledge Pipelines` card，开始把群知识编排状态沉淀为稳定 control-plane 视图。
- `Knowledge Pipeline Diagnostics` 已新增 source-seq lag / stalled 诊断：Go 可按群比较 inbox `metadata.seq` 与 checkpoint cursor，直接看出 memory / rag checkpoint 是否落后、是否在高压下停滞，并聚合到 runtime overview summary。
- `Knowledge Pipeline Diagnostics` 已新增 checkpoint age / stagnant 诊断：Go 会按群暴露 memory / rag checkpoint `age_seconds`，区分短暂 lag 与长时间未推进的 stagnant pipeline，并把 stale/stagnant target 汇总到 runtime overview。
- `Knowledge Pipeline Diagnostics` 已新增 per-group job lease freshness：Go 会按群暴露 `group_memory_extract` / `rag_ingest` stage 的 pending/active age、stale active lease、expired active lease，并把 lease 卡死直接提升到 pipeline warn/blocked。
- `Knowledge Pipeline Diagnostics` 已新增 per-dataset RAG state：Go 现在能按群输出每个 `dataset_id` 的 `rag_ingest` stage、checkpoint、lag 和 degraded 状态，避免只看到群级 `rag_checkpoint_lag_max` 却不知道是哪一个 dataset 出问题。
- observe-only QQ 群现已支持配置期 RAG dataset 绑定：Python `QQGroupConfig.ragflow_dataset_ids` 和全局 `ragflow.default_dataset_ids` 会同步到 Go `observe_target.metadata`，`Knowledge Pipeline Diagnostics` 可在 job/checkpoint 尚未生成前先显示“已配置但未启动”的 dataset。
- `rag_ingest` 成功后的稳定快照元数据现已沉到 Go checkpoint metadata，并由 `Knowledge Pipeline Diagnostics` 以结构化 `ingest_snapshot` 暴露：包括 message/document 数、seq 范围、parse 请求标记和 display name，前端不再需要自己解析原始 metadata。
- `Knowledge Pipeline Diagnostics` 已新增 per-dataset RAG index state：Go 基于 checkpoint snapshot metadata 推导 ready/missing snapshot/empty index/source lagging 状态，并聚合到 runtime overview，不调用 RAGFlow、不接管 Python RAG 策略。
- observe-only knowledge job 定期入队已迁到 Go runtime：新增 `knowledge_job_planner` worker，基于 Go observe targets 创建 `group_memory_extract` / `rag_ingest` AgentJob，并复用现有 dedupe/backpressure；Python knowledge worker 检测到 Go planner 启用后只做 lease 执行。
- 已实现 Go-owned knowledge job planner preview：`/v1/knowledge-job-planner/preview` 可在不创建 AgentJob 的情况下展示 eligible observe-only QQ 群、跳过原因、计划创建的 `group_memory_extract` / `rag_ingest` job、dedupe key、route、payload 和 dataset 绑定。
- runtime overview 已聚合 knowledge job planner preview：summary/card/detail 可直接看到计划 admission 的 observe-only QQ targets、groups、`group_memory_extract` jobs 和 `rag_ingest` jobs，仍保持只读且不创建 AgentJob。
- 已实现 Go-owned knowledge job planner readiness：`/v1/knowledge-job-planner/readiness` 聚合 preview、runtime config、runtime worker 和 Python knowledge worker heartbeat，输出 ready/reason/blockers，作为启用真实 admission 前的只读门禁。
- runtime overview 已聚合 knowledge job planner readiness：summary/card/detail 可直接看到 planner 是否 ready、blocker 数、planner enabled/running 状态和 Python knowledge worker 可用性，仍保持只读且不创建 AgentJob。
- 已实现 Go-owned knowledge job planner cutover plan：`/v1/knowledge-job-planner/cutover-plan` 输出 Go planner 与 Python legacy enqueue 的当前/目标 admission owner、required/enable/verify/rollback steps 和 blockers；runtime overview 已聚合 `Knowledge Planner Cutover`，全程只读，不改环境变量、不创建 AgentJob、不执行 group memory/RAG/AI。
- Python dashboard 已规范化 Go-owned `knowledge_job_planner_cutover_plan`：`/api/dashboard/runtime-overview` 会暴露 cutover summary 默认值、`Knowledge Planner Cutover` card 和只读 readiness/step/detail，仍不接管 admission、不创建 AgentJob、不执行 group memory/RAG/AI。
- 已实现 Go-owned Python AI worker status registry，并接入 runtime overview。
- Python AI worker status registry 已增加 Go-owned lease/fencing：新 reporter 上报 `instance_id` 和 `lease_ttl_seconds`，Go 拒绝同一 `worker_id` 活跃租约期间的异实例状态覆盖。
- Python AI workers 已增加 worker status heartbeat renewal：image / knowledge / rag_eval / outbox 长任务运行期间周期上报 `running/current_job_id`，刷新 Go worker-status lease；AgentJob lease 语义不变。

## Knowledge / Group Memory / RAG

- observe-only 群记忆抽取生命周期已迁到 Go generic jobs，Python 作为抽取 worker。
- observe-only 群知识任务 admission 已迁到 Go `knowledge_job_planner`，Python knowledge worker 保留兼容 fallback，但在 Go runtime 声明 planner 启用时不再自行入队。
- group-memory 和 RAGFlow 源消息回放已接入 Go inbox API。
- RAGFlow ingest 游标已接入 Go knowledge checkpoints。
- RAG evaluation jobs 已做成 Go-owned 生命周期记录，Python 作为 opt-in eval worker 执行离线 fixture 并回写指标。
- knowledge worker diagnostics、checkpoint dashboard 可见性，以及 job-type backlog pressure 可见性已完成。

## Proactive Runtime State

- delivery 去重、context-only 节流、drift interval mark 已迁到 Go proactive state。
- AnyAction quota、seen items、rejection cooldown、retention cleanup、background context global mark 已迁到 Go。
- proactive drift 完成态、recent runs 和 per-skill runtime state 已迁到 Go，Python 保留 workspace JSON mirror。
- proactive tick start/finish/step 审计状态已迁到 Go，Python 保留 SQLite `tick_log` / `tick_step_log` mirror 供现有 dashboard 兼容读取。
- dashboard proactive tick log 读路径已增加 Go fallback：SQLite 无匹配 tick 时读取 Go `/v1/proactive/tick-logs`、detail 和 steps；Go 查询支持 dashboard 所需分页、时间区间和排序。

## Scheduler Runtime State

- scheduler durable job snapshot、diagnostics、execution lease、job upsert/delete、completion mutation、startup recovery reconciliation 已迁到 Go。
- Python scheduler 仍负责 tick loop、cron/interval 语义、AI 执行和平台发送。

## Runtime Overview

- 已实现 Go-owned runtime overview aggregate endpoint，聚合 adapter、queue、send ledger、inbox、agent job、outbox、knowledge、observe、receiver、worker、scheduler 等诊断。
- runtime overview summary/card 已接入 selected queue provider capability：可直接展示 NATS-first 推荐阶段、provider 实现状态、多 goroutine consumer、delayed nack、external lease 和 agent_job result-ack 能力。
- dashboard 优先读取 Go aggregate，并保留必要 fallback。
- dashboard 已规范化 Go-owned runtime overview 的 `delivery_smoke_readiness` 和 `media_asset_content_diagnostics`：前端/API 可稳定读取最新 detail 与 summary 默认值，Python 只做只读展示适配，不执行发送、租约、OCR/VLM、文件解析或 AI。
- dashboard 已规范化 Go-owned runtime overview 的 control-plane plan/readiness detail：前端/API 可稳定读取 AgentJob capacity、AgentJob external lease readiness/plan、outbound cutover plan，Python 只做只读展示适配。
- dashboard 已规范化 Go-owned control audit detail：`/api/dashboard/runtime-overview` 稳定透传 `control_audit` card、`operator_approvals` 和 `control_mutations`，并补齐 summary 默认值；Python 仍只做展示适配，不记录 approval/mutation、不执行控制面变更。

## SDD / 迭代治理

- 已补充 Python AI runtime 职责边界：模型/provider、prompt/context、工具执行、Memory/RAG 算法、chunking、retrieval、Embedding/Rerank/OCR/VLM、图片生成执行、群知识沉淀、评估和快速实验继续归 Python；确定性控制面状态归 Go。
- 已沉淀 `docs/sdd/ITERATION_PROMPT.md`，明确每轮迭代必须完成 `TODO.md` 中所有未完成项，不能只提交一个未闭环切片；用户在迭代中追加的本轮要求也必须先写入 TODO 并一起闭环。
- 已新增 SDD spec index guard：`tests/test_sdd_spec_index.py` 会要求 `docs/sdd/specs/agent-gateway/000-index.md` 逐名引用所有 agent-gateway spec 文件，并补齐当前缺失索引，防止 Go 化迁移设计记录漂移。
- 已增强 Go runtime 架构守卫：`services/agent-runtime/architecture_test.go` 现在同时约束源码只能落在 `api/app/cmd/domain/infrastructure/smoke/trigger/types` 顶层根目录，防止新增随意 Go package 破坏 DDD + 六边形结构。
- 已新增 `docs/sdd/OPEN_ISSUES.md` 未解决问题总账，并用 `tests/test_sdd_governance_docs.py` 固化 TODO/DONE/BACKLOG/LIVE_CHECKS/OPEN_ISSUES 的职责边界，防止待解决问题继续堆进本轮 TODO。
- `OPEN_ISSUES.md` 已增加结构化质量守卫：`tests/test_sdd_governance_docs.py` 会校验每条 open issue 都有唯一 `OI-###`、领域、问题、影响、下一步和合法状态，避免未解决问题总账退化成散乱备注。
