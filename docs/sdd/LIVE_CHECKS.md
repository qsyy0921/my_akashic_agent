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
- [ ] 如果出现同一 worker_id 多进程抢占，后续补 worker lease/fencing，不用状态覆盖硬顶。

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
