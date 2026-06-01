# Akashic 运行态观察清单

最后更新：2026-06-01

这些项目不是新的开发任务，而是已完成切片上线后的 live smoke / 观察验证。不要再塞回 `TODO.md`。

## QQ / Telegram / Observe

- [ ] 继续验证 QQ 群实时采集质量：Go `/v1/observe-capture-diagnostics` 应显示观察群文本、图片、文件覆盖和 media content ready 状态；新文件样本出现后确认 file coverage 从 0 变为 covered，且 observe-only 不回复。
- [ ] 查看 `/v1/media-assets/content-diagnostics?asset_id=...`：真实 QQ 图片/文件应显示 `content_status=ready`；若为 `forbidden`/`unavailable`/`disabled`，应能对应到安全根目录、文件缺失或 content reader 配置问题，调用后不执行 OCR/VLM/文件解析。
- [ ] 若 Telegram `getUpdates` conflict 再次出现，检查 Go `/v1/receiver-statuses` 的 `status=suspended`、`reason=getupdates_conflict`，并确认 `/v1/receiver-leases` 没有重复 Akashic receiver；若仍冲突，排查外部 polling 进程或改 webhook。
- [ ] 做 QQ/NapCat Go adapter live send smoke：覆盖 1049511700/2365524513 双账号私聊文本、群文本、图片、文件；通过后再把对应 QQ channel alias 加入 `integrations.agent_runtime.outbound_channels` 或启用 Go local outbox worker。
- [ ] 做 QQ/NapCat cutover 前先查看 `/v1/outbound-cutover/readiness`：应同时满足 OneBot aliases 完整、delivery smoke ready、`execution_ready=true`；若 local worker 和 NATS external lease 都没执行 outbox，应出现 `outbox_execution_path_not_ready`。
- [ ] 做 QQ/NapCat cutover 前查看 `/v1/outbound-cutover/plan`：确认 `desired_execution_owner`、enable steps、verification endpoints 和 rollback steps 与本次计划一致，并确认 `side_effect=none`。
- [ ] 做 QQ/NapCat cutover 前查看 `/v1/runtime-overview` 的 `outbound_cutover_plan_*` summary 和 `Outbound Cutover` card：应与 `/v1/outbound-cutover/plan` 一致，并能直接看到当前/目标/推荐 execution owner 和 blocker 数。
- [ ] 继续观察 QQ/NapCat `message_id` 和 group upload file 字段在真实重连/重投场景的稳定性；如果 NapCat 暴露新的文件 notice 字段名，补进 `_file_meta_from_notice` 和 file-event dedupe key metadata。

## AgentJob / Worker / Queue

- [ ] 观察 Go `AgentJob` admission dedupe live 效果：重启 Python knowledge worker 后确认 `group_memory_extract` / `rag_ingest` pending 数不再按分钟无界增长。
- [ ] 启用真实 `knowledge_job_planner` 前先查看 `/v1/knowledge-job-planner/preview`：应看到本轮 observe-only QQ 群、`group_memory_extract` / `rag_ingest` 计划 job、`side_effect=none`，并确认调用后 `/v1/jobs` 未新增记录。
- [ ] 启用真实 `knowledge_job_planner` 前查看 `/v1/knowledge-job-planner/readiness`：若 preview 有 planned jobs 且 Python `knowledge` worker active，应 `ready=true`；若 planner 已启用但 runtime worker 未运行，应出现 `knowledge_job_planner_worker_not_running` blocker。
- [ ] 启用真实 `knowledge_job_planner` 前查看 `/v1/knowledge-job-planner/cutover-plan`：应显示 `current_admission_owner`、`desired_admission_owner`、enable/verify/rollback steps 和 `side_effect=none`；调用后不得新增 `/v1/jobs` 记录、不得修改环境变量、不得启动 worker 或执行 group memory/RAG/AI。
- [ ] 查看 `/v1/runtime-overview` 的 `knowledge_job_planner_preview_*` summary 和 `Knowledge Planner` card：应与 `/v1/knowledge-job-planner/preview` 的 targets / groups / total jobs 一致，且不新增 `/v1/jobs` 记录。
- [ ] 查看 `/v1/runtime-overview` 的 `knowledge_job_planner_readiness_*` summary 和 `Knowledge Planner Readiness` card：应与 `/v1/knowledge-job-planner/readiness` 的 ready/blockers/planner/worker 状态一致，且调用后 `/v1/jobs` 未新增记录。
- [ ] 查看 `/v1/runtime-overview` 的 `knowledge_job_planner_cutover_plan_*` summary 和 `Knowledge Planner Cutover` card：应与 `/v1/knowledge-job-planner/cutover-plan` 的 decision/owners/blockers 一致，且调用后仍不创建 AgentJob、不触发 Python AI worker。
- [ ] 查看 Python dashboard `/api/dashboard/runtime-overview`：应透传并规范化 Go-owned `knowledge_job_planner_cutover_plan` detail，summary 中应包含 `knowledge_job_planner_cutover_plan_*` 默认值，调用后不得创建 AgentJob、修改 env、启动 worker 或执行 group memory/RAG/AI。
- [ ] 启用 `AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED=true` 后观察 `/v1/runtime-config` 和 `/v1/runtime-workers`：应显示 `knowledge_job_planner_enabled=true` 且 `knowledge_job_planner.running=true`；Python knowledge worker 日志不再出现 legacy enqueue，但仍能 lease/execute `group_memory_extract` / `rag_ingest`。
- [ ] 查看 `/v1/job-metrics` 与 `/v1/runtime-overview.summary`：当 `group_memory_extract` / `rag_ingest` 堆积时，`pressure.by_type` 应反映 pending / active / oldest_pending_age_seconds，`agent_job_pressure_high_job_types` 应随 high pressure job type 增长。
- [ ] 查看 `/v1/runtime-overview` 的 `agent_job_worker_coverage_*` 和 `Agent Job Worker Coverage` card：当 knowledge / rag_eval / image job 堆积时，应能区分 no active worker、stale worker、failed worker 和 active worker available。
- [ ] 查看 `/v1/agent-job-capacity/plan`：当 `group_memory_extract` / `rag_ingest` / `image_generation` 堆积时，应能给出 `start_or_recover_python_worker`、`inspect_failed_worker`、`renew_or_restart_stale_worker` 或 `increase_worker_concurrency_or_prioritize_queue` 等只读建议；调用后不得新增/租约/重试 AgentJob。
- [ ] 查看 `/v1/agent-job-priority/plan`：当 `group_memory_extract` / `rag_ingest` / `rag_eval` / `image_generation` 堆积时，应按 critical/high/medium/low 输出 rank、score、action 和 blockers；调用后不得新增/租约/重试 AgentJob，不得 ack/nack MQ，不得启动 Python worker 或执行 AI。
- [ ] 查看 `/v1/runtime-overview` 与 Python dashboard `/api/dashboard/runtime-overview` 的 `agent_job_priority_*` summary、`Agent Job Priority` card 和 `agent_job_priority_plan` detail：应与 `/v1/agent-job-priority/plan` 一致，且调用后不得改变 job/MQ/worker/AI 执行状态。
- [ ] 通过 `POST /v1/operator-approvals` 记录一次针对 `agent_job_priority_plan` 或 `outbound_cutover_plan` 的 approved/rejected/revoked，再用 `GET /v1/operator-approvals?target_kind=...` 查询：应写入 `.akashic-workspace/agent-runtime/operator-approvals.json`，返回 `side_effect=runtime_state_only`，且不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 通过 `GET /v1/operator-approvals/check?target_kind=...&target_id=...` 校验上一条 approved 记录：active approved 应返回 `approved=true/reason=approval_active/side_effect=none`；missing/rejected/revoked/expired 应返回稳定 blocker，且调用后不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 通过 `POST /v1/control-mutations` 记录一次 planned/applied/failed/rolled_back 审计，再用 `GET /v1/control-mutations?approval_id=...` 查询：应写入 `.akashic-workspace/agent-runtime/control-mutations.json`，返回 `side_effect=runtime_state_only`，且不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 通过 `GET /v1/control-mutations/preflight?target_kind=...&target_id=...&action=...&operator_id=...&approval_id=...` 校验控制面变更前置条件：active approved approval 应返回 `ready=true/reason=approval_active/suggested_audit.status=planned/side_effect=none`；missing/rejected/revoked/expired/mismatch approval 应返回稳定 blockers，且调用后不得新增 approval/mutation 记录、不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 验证 control mutation policy：使用 unsupported `target_kind` 或 unsupported `action` 调用 `/v1/control-mutations/preflight` 应返回 `ready=false`、`unsupported_control_mutation_target` 或 `unsupported_control_mutation_action`，且不得继续依赖 approval 结果把任意 target/action 放行为可执行 mutation。
- [ ] 查看 `/v1/control-mutations/policy`：应返回当前 Go-owned control mutation allowlist、`side_effect=none`；按 `target_kind` 查询支持目标应只返回该目标 actions，查询 unsupported target 应返回 `allowed=false/blockers=["unsupported_control_mutation_target"]`，调用后不得创建 approval/mutation、不修改 env/config、不执行 cutover、不启动 worker、不 ack/nack MQ、不触发 AI。
- [ ] 查看 `/v1/runtime-overview` 的 `control_audit` card、`operator_approvals_*` 和 `control_mutations_*` summary：应与 `/v1/operator-approvals`、`/v1/control-mutations` 一致；调用后不得新增 approval/mutation 记录、不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 查看 Python dashboard `/api/dashboard/runtime-overview` 的 `control_audit` card、`operator_approvals`、`control_mutations` 和 summary 默认值：应与 Go `/v1/runtime-overview` 一致；调用后不得新增 approval/mutation 记录、不得修改 env/config、不得执行 cutover、不得创建/租约 AgentJob、不得 ack/nack MQ、不得触发 Python AI。
- [ ] 查看 `/v1/runtime-overview` 的 `agent_job_capacity_*` summary 和 `Agent Job Capacity` card：应与 `/v1/agent-job-capacity/plan` 的 ready/reason/blockers/summary 一致，且调用后不得启动 worker、调度、租约、重试或 ack/nack MQ。
- [ ] 查看 Python dashboard `/api/dashboard/runtime-overview`：应透传并规范化 Go-owned `agent_job_capacity_plan`、`agent_job_external_lease_readiness`、`agent_job_external_lease_plan`、`outbound_cutover_plan` detail，summary 中应包含对应默认值；调用后不得启动 worker、ack/nack MQ、创建/租约 AgentJob、创建 outbox delivery 或发送 QQ/Telegram 消息。
- [ ] 查看 `/v1/runtime-overview` 的 `media_asset_content_*` summary 和 `Media Asset Content` card：应与 `/v1/media-assets/content-diagnostics?limit=50` 的 ready/forbidden/unavailable/disabled/error 计数一致；调用后不得改变 `/content` 访问策略，不触发 OCR/VLM/文件解析。
- [ ] 查看 Python dashboard `/api/dashboard/runtime-overview`：应透传并规范化 Go-owned `delivery_smoke_readiness` 与 `media_asset_content_diagnostics` detail，summary 中应包含 `delivery_smoke_*` 和 `media_asset_content_*` 默认值；调用后不得发送 QQ/Telegram 消息、租约 outbox/AgentJob、触发 OCR/VLM/文件解析或调用 AI。
- [ ] 查看 `/v1/knowledge-pipeline-diagnostics` 与 runtime overview `Knowledge Pipelines` card：对 observe-only QQ 群应能直接看出 ready / warn / blocked，且 blocked 场景能区分 capture 问题还是 knowledge worker coverage 问题。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 `latest_source_seq` / `memory_checkpoint_lag` / `rag_checkpoint_lag_max`：真实群消息增长后，应能看到 lag 变化，并在高压积压时把 stalled target 计入 runtime overview `knowledge_pipeline_stalled_targets`。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 checkpoint `age_seconds`：当群持续有新消息但 memory / rag checkpoint 不推进时，应先看到 `*_checkpoint_stagnant`，并把 stale/stagnant target 计入 runtime overview `knowledge_pipeline_stale_checkpoint_targets` / `knowledge_pipeline_stagnant_targets`。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 stage `freshness_status` / `freshness_reason`：当 `group_memory_extract` 或 `rag_ingest` 长时间停在旧 lease 上时，应看到 `*_lease_stale` 或 `*_lease_expired`，并把目标计入 runtime overview `knowledge_pipeline_stale_active_lease_targets` / `knowledge_pipeline_expired_active_lease_targets`。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 `rag_datasets`：同一群绑定多个 RAG dataset 时，应能直接看到具体 `dataset_id` 的 lag / freshness / blocked 状态，并在 runtime overview `knowledge_pipeline_rag_dataset_*` summary 中聚合。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的配置期 dataset 可见性：当某个 observe-only QQ 群已配置 `ragflow_dataset_ids` 但还没跑出 `rag_ingest` job/checkpoint 时，应能看到 `configured_dataset_not_started`，并在 runtime overview `knowledge_pipeline_configured_rag_*` summary 中聚合。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 `ingest_snapshot`：当某个 dataset 成功完成一次 `rag_ingest` 后，应能看到最近一次 ingest 的 `message_count`、`document_count`、`start_seq`、`end_seq`、`parse_requested` 和 `display_name`，并在 runtime overview `knowledge_pipeline_rag_dataset_ingest_snapshots` 中聚合。
- [ ] 观察 `/v1/knowledge-pipeline-diagnostics` 的 `rag_index_state`：成功 ingest 且 `document_count>0` 时应为 `index_ready`；`document_count=0` 时应为 `no_index_documents`；未产生 snapshot 的配置期 dataset 应为 `no_ingest_snapshot`；source seq 增长后应能看到 `index_lagging_source_seq` 和 runtime overview `knowledge_pipeline_rag_dataset_index_*` summary。
- [ ] 观察 Go `Agent Workers` live 状态：重启 Python 主服务后确认 image/knowledge/rag_eval/outbox worker 上报 `starting` / `idle` / `running` / `stopped` 或 stale。
- [ ] 验证 worker status fencing：故意以相同 `worker_id` 启动第二个 Python AI worker 进程时，Go `/v1/agent-worker-statuses/report` 应返回 409 conflict，原实例状态不被覆盖；原实例 stopped 或 lease 过期后新实例可接管。
- [ ] 验证长任务 worker status 续租：触发一次耗时 image / knowledge / rag_eval / outbox 任务，确认 `/v1/agent-worker-statuses` 中对应 worker 的 `updated_at` 在任务运行期间持续推进，且 `lease_active=true`，不会误判 stale。
- [ ] 启用 NATS `external_lease` smoke 时观察 `/v1/queue-backend`：`external_lease.diagnostics.executed_total` 应随消费增长，`dispositions` 应反映 ack/nack/term，recent executions 不触发 Python AI job 或真实平台发送。
- [ ] 查看 `/v1/queue-backend` 的 `provider_capabilities`：当前 provider 应有 `status=selected`；NATS JetStream 应显示 `recommended=true`、`supports_concurrent_consumers=true`、`supports_delayed_nack=true`；Redis Streams / RabbitMQ 不应显示 external lease 已实现。
- [ ] 查看 `/v1/queue-topology`：应能直接看到 `outbox_delivery` 与 `agent_job` 的 `queue_source`、`execution_owner`、`ack_owner`、external lease gate state 和 blockers；调用后不得 publish/lease/ack/nack/term MQ，不得创建 AgentJob/outbox delivery，不得触发 Python AI worker 或平台发送。
- [ ] 查看 `/v1/runtime-overview` 与 Python dashboard `/api/dashboard/runtime-overview` 的 `queue_topology_*` summary、`Queue Topology` card 和 `queue_topology` detail：应与 `/v1/queue-topology` 的 work_kinds / blockers / ack owner 一致，调用后不得 publish/lease/ack/nack/term MQ，不得创建 AgentJob/outbox delivery，不得触发 Python AI worker 或平台发送。
- [ ] 查看 `/v1/runtime-overview.summary`：应包含 `queue_provider_status`、`queue_provider_recommended_phase`、`queue_provider_supports_concurrent_consumers`、`queue_provider_supports_delayed_nack`、`queue_provider_supports_external_lease`，且 queue backend card value 应带 NATS 推荐阶段。
- [ ] 查看 `/v1/queue-backend` 和 `/v1/runtime-overview.summary`：`outbox_execution_owner` / `queue_outbox_execution_owner` 应随 local worker、external lease gate 改变；`agent_job_execution_owner` / `queue_agent_job_execution_owner` 应始终明确 Python AI worker 负责执行，NATS 只在 result-ack ready 时参与确认。
- [ ] 查看 `/v1/runtime-overview` 的 `delivery_smoke_*` summary 和 `Delivery Smoke` card：应与 `/v1/delivery-smoke/readiness` 的 ready/reason/cases/blockers 一致，且调用后不得发送 QQ/Telegram 消息、租约 outbox 或触发 Python AI。
- [ ] 启用 `agent_job` NATS result-ack 前查看 `/v1/agent-job-external-lease/readiness`：应同时满足 external lease allowed、`agent_job` allowed work kind、strict lease token、`AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED=true` 和 Python worker coverage 非 danger。
- [ ] 启用 `agent_job` NATS result-ack 前查看 `/v1/agent-job-external-lease/plan`：确认 required checks、enable env、verification endpoints 和 rollback steps 与本次计划一致，且 `side_effect=none`；尤其确认 rollback 只移除 agent_job result-ack scope，不影响 outbox external lease。
- [ ] 启用 `agent_job` NATS result-ack 前查看 `/v1/runtime-overview`：`agent_job_external_lease_*` readiness summary/card 应与 `/v1/agent-job-external-lease/readiness` 一致；`agent_job_external_lease_plan_*` summary 和 `Agent Job External Lease Plan` card 应与 `/v1/agent-job-external-lease/plan` 的 decision/owners/blockers 一致，且调用后不新增或执行 AgentJob。
- [ ] 启用 NATS `external_lease` smoke 后观察 `/v1/runtime-overview`：summary 中 `queue_external_lease_*` 应与 `/v1/queue-backend.external_lease.diagnostics` 一致，`external_lease_diagnostics` card 状态应按 error/nack/term 变化。
- [ ] 压测或人为堆积 outbox 后观察 `/v1/outbox-metrics` 与 `/v1/runtime-overview`：`outbox_pressure_high_accounts`、`max_active`、`max_queued` 应随账号积压增长；该诊断只读，不应阻断真实发送。
- [ ] 启用 Go local outbox worker 账号节流后做 dry/live smoke：短时间连续发送同一账号 delivery 时，第二条应保持 queued 且不增加 attempts；其它账号 delivery 仍可被租约执行。
- [ ] 启用 NATS `external_lease` outbox smoke 并配置账号节流后，连续触发同一 `channel_kind:account_id` 的 outbox work：第一条可执行，第二条应在租约前返回 `nack/delivery_rate_limited`，Go outbox delivery 不应增加 attempts 或进入 dispatching。
- [ ] 做 external lease cutover preflight 时确认：若 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`，`/v1/queue-backend.external_lease.blockers` 应包含 `local_outbox_worker_disabled`，且 `allow_execution=false`；关闭 local worker 后再看其它 smoke gate。

## Proactive

- [ ] 观察 AnyAction quota：主动推送实际触发后确认 `.akashic-workspace/agent-runtime/proactive-state.json` 中 `anyaction_quotas.used` 增长，fallback 不放宽 quota。
- [ ] 观察 seen/rejection/cleanup：主动候选流运行后确认 `seen_items` 或 `rejection_cooldowns` 出现在 Go proactive state，并在 cleanup 后过期记录减少。
- [ ] 观察 background context：主 topic 触发后确认 `global_marks` / `bg_context_last_main_at` 出现在 Go proactive state，SQLite fallback 不让节流回退。
- [ ] 观察 drift state：启用 drift 后完成一次 `finish_drift`，确认 Go proactive state 中出现 `drift_skills`、`drift_recent_runs` 和可选 `drift_note`，workspace JSON mirror 仍写入。
- [ ] 观察 tick log：完成一次 proactive tick 后确认 Go proactive state 中出现 `tick_logs` 和可选 `tick_steps`，SQLite `tick_log` / `tick_step_log` mirror 仍写入。
- [ ] 验证 dashboard fallback：临时清空或隔离 SQLite `tick_log` mirror 后，`/api/dashboard/proactive/tick_logs` 能从 Go runtime 返回 tick list/detail/steps；runtime 不可用时仍回到 SQLite/404。

## Scheduler

- [ ] 通过 schedule tool 创建测试提醒，确认 `.akashic-workspace/agent-runtime/scheduler-jobs.json` 写入；取消后确认单任务 delete 移除对应 job。
- [ ] 创建短周期测试提醒，确认完成路径调用 Go `/v1/scheduler/jobs/{job_id}/complete`，recurring 只更新自身，one-shot 只删除自身，lease 被释放。
- [ ] 手工准备过期 recurring 和超过 grace 的 one-shot 测试 job，重启 Python scheduler 后确认 recovery reconciliation 不触发 QQ/Telegram 发送，不全量覆盖其它 job。
