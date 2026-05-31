# Akashic Go 迁移完成记录

最后更新：2026-05-31

本文件只做阶段归档，完整逐切片审查记录见 `docs/sdd/reviews/README.md` 和各 phase review 文档。

## 架构与服务边界

- Go 服务边界已统一为 `services/agent-runtime`，采用 DDD + 六边形架构分层。
- Go 负责 agent runtime/control plane，Python 负责模型调用、prompt、RAG/OCR/VLM、工具执行和快速实验。
- Go runtime 默认使用 `.akashic-workspace/agent-runtime` 文件态保存确定性状态，可用 `AKASHIC_RUNTIME_STATE_DIR` 或单项 DSN/PATH 覆盖。
- SDD 任务文档已拆分为 `TODO.md`、`DONE.md`、`LIVE_CHECKS.md`、`BACKLOG.md`：TODO 只保存本轮必须完成项，避免长期规划导致 TODO 膨胀。

## 消息、资产与发送链路

- 已实现 Go shadow audit、inbox、media asset registry、安全 content 访问和 dashboard 附件代理。
- 已实现 Go send ledger、private echo 判断和双账号防循环相关只读诊断。
- 已实现 Go outbox 生命周期、outbox event stream、delivery dispatch plan、readiness、smoke readiness 和 metrics。
- 已实现 Telegram DeliveryAdapter、QQ/NapCat OneBot HTTP/WebSocket DeliveryAdapter、adapter diagnostics、live health 和 runtime config 脱敏诊断。
- 已加入可选 Go local outbox delivery worker，默认关闭，避免未完成 cutover 时触发真实平台发送。

## 观察群与接收链路

- 已实现 Go observe target sync/list、receiver status、receiver lease、observe capture diagnostics。
- 已实现 Go inbox metrics、receiver stale 降级、recent inbox activity 推断 receiver connected。
- 已实现 Telegram 和 QQ/NapCat inbound dedupe，包含 observe-only 群文本、图片、普通群消息、私聊和群文件上传 notice。
- 已实现 inbound dedupe metrics，并接入 runtime overview/dashboard。

## AgentJob、队列与 worker 状态

- 已实现 Go generic AgentJob 生命周期、文件态持久化、dedupe_key、event stream、metrics、dashboard 恢复入口。
- 已实现 AgentJob lease token、heartbeat/renew、strict token 模式、recover-expired、queue work id 精确租约入口。
- 已完成 NATS JetStream shadow_publish、dual_read_compare、external_lease 诊断门禁和 outbox external lease smoke。
- 已实现 agent_job external lease result-ack 映射和 subject 扩容门禁，当前仍需显式 smoke/cutover flag。
- 已实现 Go-owned Python AI worker status registry，并接入 runtime overview。
- Python AI worker status registry 已增加 Go-owned lease/fencing：新 reporter 上报 `instance_id` 和 `lease_ttl_seconds`，Go 拒绝同一 `worker_id` 活跃租约期间的异实例状态覆盖。
- Python AI workers 已增加 worker status heartbeat renewal：image / knowledge / rag_eval / outbox 长任务运行期间周期上报 `running/current_job_id`，刷新 Go worker-status lease；AgentJob lease 语义不变。

## Knowledge / Group Memory / RAG

- observe-only 群记忆抽取生命周期已迁到 Go generic jobs，Python 作为抽取 worker。
- group-memory 和 RAGFlow 源消息回放已接入 Go inbox API。
- RAGFlow ingest 游标已接入 Go knowledge checkpoints。
- RAG evaluation jobs 已做成 Go-owned 生命周期记录，Python 作为 opt-in eval worker 执行离线 fixture 并回写指标。
- knowledge worker diagnostics 和 checkpoint dashboard 可见性已完成。

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
- dashboard 优先读取 Go aggregate，并保留必要 fallback。

## SDD / 迭代治理

- 已补充 Python AI runtime 职责边界：模型/provider、prompt/context、工具执行、Memory/RAG 算法、Embedding/Rerank/OCR/VLM、图片生成执行、评估和快速实验继续归 Python；确定性控制面状态归 Go。
- 已沉淀 `docs/sdd/ITERATION_PROMPT.md`，明确每轮迭代必须完成 `TODO.md` 中所有未完成项，不能只提交一个未闭环切片。
