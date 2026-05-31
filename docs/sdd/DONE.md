# Akashic Go 迁移完成记录

最后更新：2026-05-31

本文件只做阶段归档，完整逐切片审查记录见 `docs/sdd/reviews/README.md` 和各 phase review 文档。

## 架构与服务边界

- Go 服务边界已统一为 `services/agent-runtime`，采用 DDD + 六边形架构分层。
- Go 负责 agent runtime/control plane，Python 负责模型调用、prompt、Memory/RAG/chunking/retrieval、OCR/VLM、工具执行、群知识沉淀和快速实验。
- Go runtime 默认使用 `.akashic-workspace/agent-runtime` 文件态保存确定性状态，可用 `AKASHIC_RUNTIME_STATE_DIR` 或单项 DSN/PATH 覆盖。
- SDD 任务文档已拆分为 `TODO.md`、`DONE.md`、`LIVE_CHECKS.md`、`BACKLOG.md`：TODO 只保存本轮必须完成项，避免长期规划导致 TODO 膨胀。

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
- 已实现 `agent_job` external lease result-ack readiness：`/v1/agent-job-external-lease/readiness` 聚合 queue backend gate、strict lease token、runtime flag、AgentJob pressure 和 Python worker coverage，判断是否可以把 generic job 队列确认权扩展到 NATS external lease；Go 仍不执行 AI job。
- 已实现 `agent_job` external lease result-ack plan：`/v1/agent-job-external-lease/plan` 输出只读启用、验证和回滚步骤，明确 Python AI worker 继续执行模型/RAG/memory/OCR/VLM/图片任务，Go 只规划确定性 AgentJob 生命周期确认权。
- runtime overview 已聚合 `agent_job` external lease result-ack plan：summary/card/detail 可直接看到当前/目标/推荐 execution owner、决策、blocker 数和回滚步骤，仍保持只读且不 ack/nack MQ、不执行 AI job。

## 观察群与接收链路

- 已实现 Go observe target sync/list、receiver status、receiver lease、observe capture diagnostics。
- 已实现 Go inbox metrics、receiver stale 降级、recent inbox activity 推断 receiver connected。
- 已实现 Telegram 和 QQ/NapCat inbound dedupe，包含 observe-only 群文本、图片、普通群消息、私聊和群文件上传 notice。
- 已实现 inbound dedupe metrics，并接入 runtime overview/dashboard。

## AgentJob、队列与 worker 状态

- 已实现 Go generic AgentJob 生命周期、文件态持久化、dedupe_key、event stream、metrics、dashboard 恢复入口。
- 已实现 AgentJob lease token、heartbeat/renew、strict token 模式、recover-expired、queue work id 精确租约入口。
- 已完成 NATS JetStream shadow_publish、dual_read_compare、external_lease 诊断门禁和 outbox external lease smoke。
- external_lease cutover gate 已增加 local outbox worker 冲突检查：`AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 时 gate 保持 blocked，防止 state-store worker 与 NATS external lease 同时执行真实发送。
- 已实现 agent_job external lease result-ack 映射和 subject 扩容门禁，当前仍需显式 smoke/cutover flag。
- `/v1/queue-backend` 已增加 MQ provider capability matrix：明确 NATS JetStream 是当前推荐第一外部 MQ，暴露 shadow/dual-read/external-lease/agent_job result-ack、多 goroutine consumer 和 delayed nack 能力，并把 Redis Streams / RabbitMQ 标记为后续可替换 adapter 边界。
- `/v1/queue-backend` 已增加 execution owner 诊断：outbox delivery 可明确区分 `go_state_store_api`、`go_local_outbox_worker`、`nats_external_lease`；agent_job 可明确区分 Python 通过 state-store lease 执行，或 Python 执行后经 NATS result-ack 回确认。
- external_lease 已增加 Go-owned 执行诊断：`/v1/queue-backend` 可查看 ack/nack/term disposition、reason、work_kind 统计和 bounded recent executions；只读诊断不执行 Python AI job。
- runtime overview 已聚合 external_lease 执行诊断：summary/card 可直接查看执行总数、错误数、ack/nack/term 分布，原始 queue backend detail 仍保留。
- runtime overview summary 已聚合 queue execution owner 字段：`queue_outbox_execution_owner` 和 `queue_agent_job_execution_owner` 能直接说明当前执行边界，避免把 NATS result-ack 误解为 Go 执行 AI job。
- 已实现 Go-owned `AgentJob` pressure 诊断：`/v1/job-metrics` 可按 `job_type` 查看 pending / leased / running / active / oldest_pending_age，并标记 high pressure；runtime overview 也聚合了 `agent_job_pressure_*` summary 与 `Agent Job Pressure` card。
- runtime overview 已新增 `Agent Job Worker Coverage`：把 Go-owned `AgentJob` pressure 与 Python worker heartbeat / stale / failed 状态关联起来，直接解释 backlog 是否由无 active worker、stale worker 或 failed worker 导致，不改变 Python 执行边界。
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

## SDD / 迭代治理

- 已补充 Python AI runtime 职责边界：模型/provider、prompt/context、工具执行、Memory/RAG 算法、chunking、retrieval、Embedding/Rerank/OCR/VLM、图片生成执行、群知识沉淀、评估和快速实验继续归 Python；确定性控制面状态归 Go。
- 已沉淀 `docs/sdd/ITERATION_PROMPT.md`，明确每轮迭代必须完成 `TODO.md` 中所有未完成项，不能只提交一个未闭环切片；用户在迭代中追加的本轮要求也必须先写入 TODO 并一起闭环。
