# Akashic 运行态观察清单

最后更新：2026-05-31

这些项目不是新的开发任务，而是已完成切片上线后的 live smoke / 观察验证。不要再塞回 `TODO.md`。

## QQ / Telegram / Observe

- [ ] 继续验证 QQ 群实时采集质量：Go `/v1/observe-capture-diagnostics` 应显示观察群文本、图片、文件覆盖和 media content ready 状态；新文件样本出现后确认 file coverage 从 0 变为 covered，且 observe-only 不回复。
- [ ] 若 Telegram `getUpdates` conflict 再次出现，检查 Go `/v1/receiver-statuses` 的 `status=suspended`、`reason=getupdates_conflict`，并确认 `/v1/receiver-leases` 没有重复 Akashic receiver；若仍冲突，排查外部 polling 进程或改 webhook。
- [ ] 做 QQ/NapCat Go adapter live send smoke：覆盖 1049511700/2365524513 双账号私聊文本、群文本、图片、文件；通过后再把对应 QQ channel alias 加入 `integrations.agent_runtime.outbound_channels` 或启用 Go local outbox worker。
- [ ] 继续观察 QQ/NapCat `message_id` 和 group upload file 字段在真实重连/重投场景的稳定性；如果 NapCat 暴露新的文件 notice 字段名，补进 `_file_meta_from_notice` 和 file-event dedupe key metadata。

## AgentJob / Worker / Queue

- [ ] 观察 Go `AgentJob` admission dedupe live 效果：重启 Python knowledge worker 后确认 `group_memory_extract` / `rag_ingest` pending 数不再按分钟无界增长。
- [ ] 观察 Go `Agent Workers` live 状态：重启 Python 主服务后确认 image/knowledge/rag_eval/outbox worker 上报 `starting` / `idle` / `running` / `stopped` 或 stale。
- [ ] 验证 worker status fencing：故意以相同 `worker_id` 启动第二个 Python AI worker 进程时，Go `/v1/agent-worker-statuses/report` 应返回 409 conflict，原实例状态不被覆盖；原实例 stopped 或 lease 过期后新实例可接管。
- [ ] 验证长任务 worker status 续租：触发一次耗时 image / knowledge / rag_eval / outbox 任务，确认 `/v1/agent-worker-statuses` 中对应 worker 的 `updated_at` 在任务运行期间持续推进，且 `lease_active=true`，不会误判 stale。
- [ ] 启用 NATS `external_lease` smoke 时观察 `/v1/queue-backend`：`external_lease.diagnostics.executed_total` 应随消费增长，`dispositions` 应反映 ack/nack/term，recent executions 不触发 Python AI job 或真实平台发送。
- [ ] 查看 `/v1/queue-backend` 的 `provider_capabilities`：当前 provider 应有 `status=selected`；NATS JetStream 应显示 `recommended=true`、`supports_concurrent_consumers=true`、`supports_delayed_nack=true`；Redis Streams / RabbitMQ 不应显示 external lease 已实现。
- [ ] 查看 `/v1/runtime-overview.summary`：应包含 `queue_provider_status`、`queue_provider_recommended_phase`、`queue_provider_supports_concurrent_consumers`、`queue_provider_supports_delayed_nack`、`queue_provider_supports_external_lease`，且 queue backend card value 应带 NATS 推荐阶段。
- [ ] 查看 `/v1/queue-backend` 和 `/v1/runtime-overview.summary`：`outbox_execution_owner` / `queue_outbox_execution_owner` 应随 local worker、external lease gate 改变；`agent_job_execution_owner` / `queue_agent_job_execution_owner` 应始终明确 Python AI worker 负责执行，NATS 只在 result-ack ready 时参与确认。
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
