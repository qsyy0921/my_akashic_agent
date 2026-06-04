# Akashic Go 迁移剩余清单

最后更新：2026-06-04

这份文档只回答一个问题：

> 现在还有哪些东西**没有改完**，或者**虽然代码已经有了，但还没有真正切换到 Go 执行**？

为了避免讨论跑偏，本文把剩余工作分成三类：

1. **已实现但未完成 cutover**
2. **Go control plane 已有，但真实执行器还没做**
3. **明确保留在 Python，不属于 Go 迁移未完成**

相关权威来源：

- `docs/sdd/DONE.md`
- `docs/sdd/OPEN_ISSUES.md`
- `docs/sdd/LIVE_CHECKS.md`

## 一、已实现但未完成 cutover

这些能力的 Go 代码、诊断接口、只读 plan/readiness 大体已经有了，
但还没有完成真实运行切换或全量 live smoke。

### 1. QQ / NapCat 真实发送 cutover

当前状态：

- Go OneBot/NapCat `DeliveryAdapter` 已完成。
- `runtime-config`、`delivery-adapters/health`、`delivery-smoke/readiness`、
  `outbound-cutover/readiness`、`outbound-cutover/plan` 都可用。
- 双 QQ 账号私聊文本真实 smoke 已通过：
  - `1049511700 -> 2365524513`
  - `2365524513 -> 1049511700`

还没完成的部分：

- 群图片真实 smoke 通过
- 群文件真实 smoke 通过
- rich media 从 `text_only` 扩大到全量 execution owner

最新 live 结果：

- 群文本真实 smoke 已通过：`1049511700 -> group:27234224`
- 群图片 / 群文件当前都被 NapCat / QQ rich media 上传阻塞，错误为
  `rich media transfer failed`
- 原生 NapCat rich-media 对照现在已在 `27234224`、`3219982`、`164369633`
  三个群上重复得到相同失败，说明当前 blocker 更接近这次 NapCat / QQ
  rich-media 会话本身，而不是单一群配置差异
- 简单 `docker restart napcat` 后，对 `3219982` 的原生 rich-media 对照
  仍然失败，说明仅刷新容器不足以修复当前会话
- 第二账号 `2365524513` 的验证进一步把 blocker 拆开：
  - `file` 已能在原生 NapCat 与 Akashic 两条路径成功发送
  - `image` 仍在原生 NapCat 与 Akashic 两条路径失败
  说明 image 不是单账号或 Akashic adapter 问题，而第一账号的 file 失败
  则是 account-session-specific
- 当前 runtime 已新增并验证 account-kind gate 与
  account-conversation-kind gate：
  - 全局仍保持 `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
  - 第二账号通过
    `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT=2365524513=text|file`
    额外放开了 `file`
  - 第一账号通过
    `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE=1049511700/private=text|file`
    额外放开了 `private file`
  - 第一账号 `group file` 仍被 gate 保持阻塞
- 在这一运行态下，第二账号 `group file` outbox 已由 Go local worker
  自动成功；第一账号 `private file` outbox 也已由 Go local worker
  自动成功；第一账号 `group file` outbox 仍保持 `queued`
- 当前还新增了 repo-owned live smoke
  `scripts/verify-go-outbox-scope-live.ps1`，会直接复核：
  - 第一账号 `private file` 自动成功
  - 第二账号 `group file` 自动成功
  - 第一账号 `group file` 仍 gated
  - 第一账号 `private image` 仍 gated
- 本轮还修复了 repo-local launcher 缺失
  `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` 的问题；修复并重启 runtime 后，
  `scripts/verify-go-outbox-scope-live.ps1` 与统一 goal verifier 已再次确认
  第二账号 `group file` 的自动成功不是偶然结果，而是正确走到了
  `qq_2365524513` alias
- 新增 private 原生对照后，rich-media 路由边界又被切细了一层：
  - `1049511700 -> 2365524513` private image 原生失败，private file 原生成功
  - `2365524513 -> 1049511700` private image 原生失败，private file 原生成功
  - 第一账号 Akashic private file 也已成功，private image 则与 native 一样失败
- 新增原生对照 `scripts/run-napcat-native-rich-media-smoke.ps1` 后，
  直接绕开 Akashic 对同一 NapCat WebSocket 执行 streamed
  `upload_file_stream -> send_group_msg/upload_group_file`，仍返回同样的
  `rich media transfer failed`
- 观察群静默本身已经不是剩余迁移项：当前同步到 runtime 的 QQ observe
  targets 都是 `reply_allowed=false`，并且 Go delivery dispatch 已对 exact-
  match observe-only group route 做硬阻断；真实 `qq:1049511700:group:27234224`
  readiness 已返回 `observe-only target does not allow replies`。也就是说，
  “观察群只观察、不回群消息”现在已经是 runtime invariant，而不是仅靠配置约定
- text-only Go outbox owner 已完成真实验证：
  - `/v1/queue-backend` 当前为
    `outbox_execution_owner=go_local_outbox_worker` /
    `outbox_execution_scope=text_only`
  - `/v1/outbound-cutover/readiness` 当前为
    `execution_ready=true/execution_owner=go_local_outbox_worker`
  - 新的私聊文本 outbox event 已真实经过
    `queued -> leased -> succeeded`
  - 新的群图片/文件 outbox event 在连续轮询中保持
    `queued/attempts=0`
  - Python compatibility outbox worker 当前在
    `/v1/agent-worker-statuses` 中显示
    `reason=go_runtime_outbox_worker_active`

当前 blocker：

- rich media 原生对照现在已证明：
  - `image` 在 group/private 两种路由上都失败，因此当前不能做任何 image cutover
  - `file` 不是全局失败，而是第一账号 group route 失败
- 当前本机还额外启用了全局 QQ group send manual toggle：
  - Go: `AKASHIC_QQ_GROUP_SEND_ENABLED=false`
  - Python: `group_send_enabled=false`
  这意味着当前任何 `qq/group` route 都不会实际发送；如果后续要恢复群发验证，必须先手动打开开关，再做群级 live smoke。
- 当前已不再局限于 `text_only`：第二账号 `file` 已可通过
  account-kind gate 进入 Go default owner，第一账号 `private file`
  已可通过 account-conversation-kind gate 进入 Go default owner，但
  `image` 与第一账号 `group file` 仍不能放开
- 下一步依赖两类变化：
  - 人工重登或替换第一账号 `1049511700` 的 group rich-media 会话
  - image 的跨账号原生失败需要先解决，之后才能考虑是否扩大到全局
    `image/file`

当前已保持的部分启用路径：

- 当前本机运行态已经启用
  `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`
  和 `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text`
  以及
  `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT=2365524513=text|file`
  与
  `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE=1049511700/private=text|file`
- 也就是说，Go local outbox worker 现在已经是这台机器上的真实文本
  execution owner，而不是只做过一次临时 smoke

这意味着：

- Go 已经能真实发一部分 QQ 消息；
- 当前 rich media blocker 已经确认在 NapCat / QQ session 平台侧，而不是
  Akashic adapter staging 层；
- `Media Asset Content` dashboard/runtime parity 已不再是剩余项：当前 fallback
  已能稳定保留 content diagnostics 与 recovery/preflight 链接，剩余的 media
  gap 已收敛到 private-source fetch / background repull / post-recovery AI
  enrichment；
- `go_local_outbox_worker` 这条 execution-owner 路径已经被真实验证可行；
- “全局文本 + 第二账号 file + 第一账号 private file”的 execution-owner
  分层已经存在并通过 live 验证；
- 当前剩下的不是 Go text owner 缺失，而是 rich media 平台 blocker 未解除。

### 2. Telegram 后台链路

当前状态：

- Go Telegram adapter 代码已存在。
- Telegram Desktop 已安装并可人工登录。
- 当前还新增了 repo 内验证入口
  `scripts/verify-telegram-backend.ps1`，可直接从 live runtime +
  当前环境复核 token 是否存在、receiver 是否出现，以及是否能够执行
  `getMe`
- 该验证入口现已增强到可同时证明：
  - `config.toml` 已声明 `[channels.telegram]`
  - `token = "${TELEGRAM_BOT_TOKEN}"`
  - 当前 repo `.env`、`~/.codex/.env` 和本进程环境都没有 Telegram token
- 当前还新增了统一 goal 审计入口
  `scripts/verify-go-migration-goal.ps1`，可把 Telegram backend 状态与
  QQ outbox scope、knowledge planner 和其它残留分类一起输出为单个 JSON
  ；并可通过 `-IncludeOutboxScopeSmoke` 把真实 Go outbox scope smoke 合并进同一份证据
- 当前还新增了 repo-owned route-matrix verifier：
  `scripts/verify-qq-cutover-route-matrix.ps1`
- 该 verifier 已在本机真实通过，并把当前 route 直接拆成：
  - `go_execution_owner_scope`
  - `currently_sendable_routes`
  - `policy_blocked_routes`
  - `platform_blocker_routes`
- runtime overview dashboard 现在也已直接展示同一份
  `qq_cutover_route_matrix` read-model，主 panel 可直接读出
  `go scope/currently sendable/policy blocked/platform blockers`，不再需要只靠
  单独脚本或 raw JSON 理解当前 QQ cutover 边界。
- 这意味着当前剩余 gap 不再是“必须人工拼 allowed/unresolved routes 才能解释 QQ 边界”，
  而是 rich-media blocker 何时恢复以及是否允许重新开启群发策略。
- 该统一 verifier 现在还会默认检查 observe-only 静默边界：
  - 当前是否仍存在 enabled 且 `reply_allowed=false` 的 QQ group observe target
  - 对真实观察群 route 的 readiness 是否仍被
    `observe-only target does not allow replies` 阻断
  - QQ private readiness 是否仍正常

还没完成的部分：

- `TELEGRAM_BOT_TOKEN` 未配置
- 因此 Telegram bot backend 没有完成 `getMe` / receiver / 收发 smoke

注意：

- Desktop 登录不等于 backend ready。
- 当前 `/v1/runtime-config` 仍显示 `telegram_token_configured=false`，`/v1/receiver-statuses` 也仍只有 2 个 QQ receiver。
- 当前 blocker 已经被收敛成更强证据：Telegram 不是“链路坏了但原因不明”，而是“通道已声明，但 token 仍停留在 `${TELEGRAM_BOT_TOKEN}` 占位符且没有注入到本地 runtime”。
- 这属于“外部凭证未到位导致的未完成 cutover”，不是 Go 侧完全没实现。

### 3. AgentJob external lease result-ack

当前状态：

- readiness / plan / diagnostics 已完成
- strict lease token、worker coverage、rollback 约束都已有只读门禁
- 当前还新增了 repo-owned verifier：
  - `scripts/verify-agent-job-external-lease.ps1`
- 当前又新增了 repo-owned temp NATS smoke verifier：
  - `scripts/verify-agent-job-external-lease-nats-smoke.ps1`
- 当前还新增了 repo-owned isolated cutover preflight verifier：
  - `scripts/verify-agent-job-external-lease-cutover-preflight.ps1`
- 当前还新增了 read-only approval-bound endpoint：
  - `GET /v1/agent-job-external-lease/preflight`
- 当前还新增了 repo-owned launcher preflight verifier：
  - `scripts/verify-agent-job-external-lease-launcher-preflight.ps1`
- 当前还新增了 canonical launcher bundle endpoint 与 repo-owned live verifier：
  - `GET /v1/agent-job-external-lease/launcher-bundle`
  - `scripts/verify-agent-job-external-lease-launcher-bundle.ps1`
- 该 verifier 已在本机真实通过：
  - `TestExternalLeaseNATSSmokeAgentJobDuplicateTerminalAck`
  - `TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow`
- isolated preflight 现在也已在本机真实通过，并固定证明：
  - 无 external-lease flags 时仍是 local provider +
    `python_ai_worker_state_store_lease`
  - 满足 temp NATS、explicit result-ack flags、active `knowledge` worker
    coverage 后，`agent_job_execution_owner`、queue topology `ack_owner`、
    readiness 和 plan 会一起翻到 ready
  - 同一个 preflight 路径现在还会稳定区分：
    - `agent_job_external_lease_plan_not_ready`
    - `missing_approval_id`
    - `agent_job_external_lease_preflight_ready`
- launcher preflight 现在也已在本机真实通过，并固定证明：
  - repo-local `scripts/start-agent-runtime.ps1` 已能显式注入 result-ack
    所需 flags
  - 通过 launcher 起的隔离 temp runtime 会把这些 flags 暴露到
    `/v1/runtime-config`
  - queue backend / queue topology / approval-bound preflight 会一起进入
    `repo_owned_launcher_external_lease_preflight_live_verified`
- launcher bundle 现在也已在本机真实通过，并固定证明：
  - blocked runtime 下 bundle 会稳定返回 exact launcher parameters、
    environment overrides、`QueueDSN` external input 和 verification steps
  - 用 bundle 动态拼出来的 `start-agent-runtime.ps1` 参数可以把 temp runtime
    拉到 promoted state
  - queue backend / queue topology / runtime-overview / approval-bound preflight
    会一起进入
    `repo_owned_launcher_external_lease_bundle_live_verified`
- 当前又新增了 live runtime cutover diff endpoint 与 repo-owned live verifier：
  - `GET /v1/agent-job-external-lease/cutover-diff`
  - `scripts/verify-agent-job-external-lease-cutover-diff.ps1`
- cutover diff 现在已在本机真实通过，并固定证明：
  - 当前 8780 runtime 仍明确 blocked，drift 直接列出缺失的
    external-lease/result-ack flags、`QueueDSN`、queue provider/mode 和
    `agent_job execution/ack owner`
  - 用同一 canonical bundle 带起的 temp NATS/runtime 会先表现为
    `zero config drift + agent_job_worker_coverage_blocked`
  - knowledge worker heartbeat 之后，cutover diff 变为
    `agent_job_external_lease_cutover_diff_ready`
- 现有 Go smoke 里的固定历史时间也已修正，不再因日期推进把 flow smoke 自己
  的 lease 判成 expired
- 它现在会直接给出当前 turn 的：
  - readiness / plan
  - runtime flags
  - queue backend selected provider
  - queue topology `agent_job` owner
  - runtime overview summary
  - blocker buckets（configuration / execution_owner / smoke / worker_coverage）
- 当前还新增了 repo-owned queue-topology verifier：
  - `scripts/verify-queue-topology-boundary.ps1`
- 该 verifier 已在本机真实通过，并证明：
  - `/v1/queue-topology`、`/v1/queue-backend` 与 `/v1/runtime-overview.summary`
    对 provider recommendation、outbox owner、agent_job owner、ack owner 和
    external-lease-ready 的描述保持一致
- 这意味着当前 queue topology 剩余问题已不再是可见性/read-model 缺口，
  而是 external-lease / result-ack 的 production cutover 何时真正切换

还没完成的部分：

- 真实生产 result-ack cutover
- 对应长期运行 runtime 的 flags / owner 切换 smoke 和最终启用

当前 live 结论已经更窄：

- 当前不是 worker coverage blocker
- 当前也不再是“repo-owned smoke 不存在或不稳定” blocker
- 当前也不再是“repo-owned cutover preflight 语义不明确” blocker
- 当前也不再是“launcher contract 缺失或 bundle 无法 reproducible 带起 preflight” blocker
- 当前是：
  - `provider=local`
  - external queue 未配置
  - strict lease token 未启用
  - result-ack scope flag 未启用
  - runtime 里 duplicate/flow smoke flags 仍未注入
  - current owner 仍是 `python_ai_worker_state_store_lease`

本质上：

- Go 已能规划和诊断；
- Python 仍是真正执行 AI job 的 owner；
- NATS result-ack 还没正式接管确认权。

### 4. Knowledge planner cutover

当前状态：

- Go `knowledge_job_planner` 的 preview / readiness / cutover plan 已有
- 相关 runtime overview 已聚合
- 本机运行态已经通过 `scripts/start-agent-runtime.ps1 -EnableKnowledgeJobPlanner ...` 启用 Go planner
- 当前 `/v1/knowledge-job-planner/cutover-plan` 已返回
  `decision=ready/current_admission_owner=go_runtime_knowledge_job_planner`
- Python knowledge worker 已继续 lease/execute 新 bucket jobs，且启用后未再产生新的 per-minute legacy enqueue 记录
- 当前还新增了 repo 内验证入口
  `scripts/verify-knowledge-planner-cutover.ps1`，可直接从 live endpoint +
  当前日志复核 planner enabled/running、recent planner buckets、Go-created
  jobs 和 Python `skip legacy enqueue` 证据
- 当前还新增了统一 goal 审计入口
  `scripts/verify-go-migration-goal.ps1`，会复用上述验证结果，并与
  outbox/Telegram/残留分类一起输出
- 统一 goal verifier 现在还会直接透传 `qq_cutover_route_matrix`，
  不再只保留 `allowed_routes + unresolved_routes` 的粗粒度摘要。

还没完成的部分：

- 长时间运行下的持续观察仍然留在 `LIVE_CHECKS.md`

### 6. Goal verification runbook

当前状态：

- 已有 repo-owned 统一验证入口
  `scripts/verify-go-migration-goal.ps1`
- 它会聚合：
  - QQ outbox execution owner / scope
  - 可选 Go outbox scope live smoke
  - 可选原生 NapCat rich-media probe
  - Telegram token / receiver / `getMe` 状态
  - knowledge planner 当前健康状态
  - agent_job external lease / scheduler / proactive / autoscaling /
    dashboard fallback / Python-owned AI runtime 的当前分类

还没完成的部分：

- 这不是新的 cutover 能力，而是新的 goal 证据入口
- rich-media 是否恢复、Telegram 是否配置 token，仍然需要后续 live 状态变化

### 7. Python worker status restart takeover

当前状态：

- Go worker-status registry、lease/fencing、heartbeat renewal 都已经存在。
- 本轮又补齐了 stale-instance 受控 takeover：
  - Go `/v1/agent-worker-statuses/report` 支持显式
    `replace_existing_instance_id`
  - Python reporter 只会在 `HTTP 409`、旧 `instance_id` 可解析且旧 PID
    已退出时重试一次 takeover
- 当前还新增了 repo-owned live verifier：
  - `scripts/verify-agent-worker-status-restart.ps1`
- 当前还新增了 repo-owned temp-runtime fencing live smoke：
  - `scripts/verify-agent-worker-status-fencing-live-smoke.ps1`
- 当前又新增了 repo-owned temp-runtime heartbeat renewal live smoke：
  - `scripts/verify-agent-worker-status-heartbeat-live-smoke.ps1`
- 当前又新增了 repo-owned stale cleanup live smoke：
  - `scripts/verify-agent-worker-status-cleanup-live-smoke.ps1`
- 当前又新增了 repo-owned startup prune live smoke：
  - `scripts/verify-agent-worker-status-startup-prune-live-smoke.ps1`
- 该 verifier 已在真实 `uv run python main.py` 重启后通过，并证明：
  - fresh `logs/bot.log` 没有新的
    `agent worker status lease conflict`
  - 没有新的 `agent worker status report failed`
  - `/v1/agent-worker-statuses` 里 `image/knowledge/outbox`
    三个 active worker 的 `instance_id` 都切到了当前 Python 主进程 PID
- 新 fencing smoke 也已在隔离 temp runtime 真实通过，并证明：
  - 同一 `worker_id` 的第二实例默认返回 `HTTP 409`
  - 冲突文本包含当前 `existing_instance_id`
  - 只有显式匹配 `replace_existing_instance_id` 时 takeover 才成功
  - API 视图和 `agent-worker-statuses.json` 都切到新实例
- 新 heartbeat renewal smoke 也已在隔离 temp runtime 真实通过，并证明：
  - 同一 `worker_id + instance_id` 的连续 `running` heartbeat 都返回 `202`
  - `updated_at` 与 `lease_until` 会随 heartbeat 持续推进
  - final record 保持 `lease_active=true`、`stale=false`
  - API 视图与 `agent-worker-statuses.json` 的最终 heartbeat 一致
- 新 stale cleanup smoke 也已在隔离 temp runtime 与当前 live runtime 真实通过，并证明：
  - `POST /v1/agent-worker-statuses/cleanup-stale` 只删除超过 stale threshold 的 heartbeat 残留
  - active worker 不会被误删
  - temp `agent-worker-statuses.json` 与 API 视图会同步删除 stale 记录
  - 当前 live runtime 原先遗留的 stale `akashic-python-worker` 也已被安全清掉，当前 `totals.stale=0`
- 新 startup prune smoke 也已在隔离 temp runtime 真实通过，并证明：
  - runtime restart 时不会把历史 stale `agent-worker-statuses` 再次载回内存
  - repository load 会直接删除 stale heartbeat residue
  - active worker record 会被保留
  - 当前 8780 live runtime 重启后仍保持 `agent_workers_stale=0`

还没完成的部分：

- 不再缺 worker-status lease 或 stale-cleanup 语义层的独立 live 回归；剩余 gap 已收敛到真实业务任务的 executor 能力
- 不再缺 runtime restart 后 stale worker residue 回灌这一类 false blocker

这意味着：

- Python 主进程重启后的 `agent worker status lease conflict`
  已不再是当前本机 Go 迁移 blocker；
- 默认 fencing / 受控 takeover / heartbeat renewal / stale cleanup 语义本身都已被 repo-owned smoke 固定下来；
- 剩下的是 worker control executor 与业务任务执行面，而不是 restart / fencing / renewal 语义缺口。

### 5. Scheduler cutover 的最终运行验证

当前状态：

- Go 已持有 scheduler snapshot、lease、diagnostics、upsert/delete、complete
- 统一 goal verifier 现在已会真实调用：
  - `/v1/scheduler/jobs`
  - `/v1/scheduler/leases`
  - `/v1/scheduler/diagnostics`
  并额外通过 `.\scripts\verify-scheduler-runtime-live-smoke.ps1`
  启动隔离 temp `agent-runtime`，真实验证：
  - `scheduler-jobs.json` 的 single-job upsert/delete 持久化；
  - lease-fenced `POST /v1/scheduler/jobs/{job_id}/complete` 的 one-shot delete
    与 recurring reschedule；
  - Python `SchedulerService.load_and_recover()` 对 Go scheduler state 的
    recurring misfire 推进与 expired one-shot 删除；
- 当前 turn 已把 scheduler 归类成
  `go_control_plane_mutation_and_recovery_live_verified`

剩余如果要继续做，只是：

- schedule tool 在当前 8780 runtime 上的 operator UX smoke；
- reminder 文案或平台发送层面的流程观察。

这些不再是 Go scheduler state/control-plane 迁移 blocker。

### 6. Proactive live verification 的剩余项

当前状态：

- Go 已持有 proactive quota、seen/rejection、cleanup、bg-context、drift
  state、tick log state。
- 当前又新增了 repo-owned isolated live smoke：
  `scripts/verify-proactive-runtime-flow-live-smoke.ps1`
- 该 smoke 已在 temp runtime 真实通过，并固定证明：
  - deliveries duplicate/count
  - seen-items normalized hit
  - rejection cooldown
  - cleanup 对 deliveries / seen / context-only / rejection 的过期清理
  - anyaction quota snapshot + increment
  - drift mark / finish / summary / skill-state
  - bg-context global mark
  - tick-log start / step / finish / list / detail / steps
  - 以及上述状态在 `proactive-state.json` 中的持久化
- 统一 goal verifier 现在已会真实调用：
  - `/v1/proactive/tick-logs`
  - `/v1/proactive/drift/summary`
  - `/v1/proactive/anyaction/quota`
  - `/v1/proactive/bg-context/main/last`
  - `/v1/proactive/context-only/last`
  - dashboard `/api/dashboard/proactive/tick_logs`
- 当前 turn 已能把 proactive 归类成
  `go_state_live_verified_with_dashboard_read_model`。
- 当前 turn 还会把该 repo-owned isolated smoke 并入
  `proactive_runtime_flow_smoke`，因此 proactive 已不再只是“当前 live
  snapshot 可读”，而是 deterministic state flow 也有 current-turn 强证据。
- 当前又新增了 repo-owned dashboard fallback verifier：
  `scripts/verify-dashboard-proactive-tick-logs-boundary.ps1`
- 该 verifier 已在隔离 temp runtime + 空 SQLite dashboard workspace 中真实
  通过，并固定证明：
  - dashboard `/api/dashboard/proactive/tick_logs` list/detail/steps 会回退到
    Go runtime tick state
  - SQLite `tick_log` mirror 在读取前后都保持 0，不会被 fallback 读路径反向写入

还没完成的部分：

- seen/rejection/cleanup/background-context/drift 的业务流触发型 live 观察
- runtime 不可用时回退到 SQLite/404 的人工场景回归

也就是说：

- proactive 不再是“没验证”；
- 剩下的是业务触发型长期观察与 runtime-unavailable 的人工 fallback 回归，
  而不是缺少 Go state/control plane 或缺少 repo-owned deterministic flow smoke。

### 7. Worker control executor boundary

当前状态：

- `agent_job capacity / priority` 两条只读 plan 已 live 可用；
- `control mutation policy` 已 allowlist：
  - `agent_job_capacity: apply, rollback`
  - `agent_job_priority: apply, rollback`
- 当前还新增了 repo-owned verifier：
  - `scripts/verify-worker-control-executor-boundary.ps1`
- 当前又新增了 repo-owned `control audit` verifier：
  - `scripts/verify-control-audit-boundary.ps1`
- 该 verifier 已在本机真实通过，并证明：
  - capacity/priority plan 与 runtime overview summary 一致；
  - priority 仍是 `manual_only`，worker control owner 仍是 `python`；
  - runtime workers 里仍没有 autoscaling / concurrency / priority executor
    candidate；
  - 当前结论是
    `go_control_plane_live_verified_without_worker_control_executor`
- 新的 `control audit` verifier 也已在本机真实通过，并证明：
  - `operator approvals` / `control mutations` 两个 ledger 会落盘到
    runtime state；
  - `/v1/operator-approvals`、`/v1/operator-approvals/check`、
    `/v1/control-mutations`、`/v1/control-mutations/preflight` 与
    `/v1/control-mutations/policy` 的 side-effect 边界稳定；
  - `/v1/runtime-overview` 的 `control_mutation_policy` 与 `control_audit`
    summary/card/detail 会和 ledger state 保持一致；
  - 当前结论是
    `control_audit_boundary_live_verified`
- 当前又新增了 repo-owned dashboard `control audit` boundary verifier：
  - `scripts/verify-dashboard-control-audit-boundary.ps1`
- 该 verifier 已在本机真实通过，并证明：
  - dashboard `/api/dashboard/runtime-overview` 的
    `control_mutation_policy` summary / top-level detail / card
    与 Go `/v1/runtime-overview` 当前 turn 一致；
  - dashboard `/api/dashboard/runtime-overview` 的
    `control_audit` card、top-level `operator_approvals` /
    `control_mutations` 和 summary 默认值与 Go runtime overview 当前 turn 一致；
  - unified goal verifier 现在对这两条 dashboard read-model 直接消费 live
    verifier 输出，不再只靠 panel 静态字符串探针

这意味着：

- 剩余 gap 不再是“capacity/priority 只有 spec 没证据”；
- 剩余 gap 也不再是“approval / audit / preflight 只有 handler 单测，没有
  repo-owned live 证据”；
- 当前 gap 是：control plane 与 control-audit 边界都已验证，但真实 executor
  还没做。

## 二、Go control plane 已有，但真实执行器还没做

这类能力不是没设计，而是目前只有：

- diagnose
- readiness
- plan
- audit
- approval

没有真正自动执行变更。

### 1. capacity / priority / cutover 的真实 mutation executor

当前状态：

- `/v1/agent-job-capacity/plan`
- `/v1/agent-job-priority/plan`
- `/v1/outbound-cutover/plan`
- operator approval / control mutation audit / policy
- `scripts/verify-worker-control-executor-boundary.ps1`
- `scripts/verify-control-audit-boundary.ps1`

都已经具备。

还没做：

- 自动调并发
- 自动启动/恢复 worker
- 自动修改配置
- 自动执行 cutover

原因很明确：

- 这些动作一旦自动化，必须先绑定 approval、audit、allowlist、rollback、
  限流和 side-effect 边界。

### 2. Python worker autoscaling / priority scheduling executor

当前状态：

- Go 能看到 pressure、worker coverage、priority plan
- 当前 live verifier 也已证明 runtime workers 里还没有
  autoscaling / concurrency / priority executor candidate

还没做：

- 根据这些信号真实拉起/缩容 Python worker
- 对 job 类型做真实优先级调度

### 3. Media recovery 的平台私有源统一执行器

当前状态：

- Go 已支持 HTTP/HTTPS media cache recovery executor
- preflight / approval / audit / recovery endpoint 都已有
- 新增了 repo-owned live verifier `scripts/verify-media-recovery-boundary.ps1`
- 新增了 repo-owned isolated live smoke
  `scripts/verify-media-asset-content-recovery-live-smoke.ps1`
- 新增了 repo-owned dashboard/runtime parity verifier
  `scripts/verify-dashboard-media-asset-content-recovery-boundary.ps1`
- 当前 live verifier 已证明：
  - sampled assets 全为 `file://` observe-only QQ group 上传镜像
  - `content_diagnostics` 为 `forbidden=120/120`
  - sample recovery plan 为 `media_asset_content_recovery_fix_content_roots`
  - sample preflight 为 `missing_approval_id/executor_scope=operator_runtime_config`
  - 当前 registry 没有 HTTP/HTTPS live candidate
  - dashboard recovery-plan proxy 正常，dashboard runtime overview 现在也已透出
    `media_asset_content_recovery` summary/card/detail，且 fallback 不再退回
    `unknown/muted`
- 当前 isolated executor smoke 已证明：
  - HTTP/HTTPS source 在正式 recovery 前不会被下载
  - dry-run 不写 cache
  - 正式 recovery 会写本地 cache、更新 registry metadata
  - `/v1/media-assets/{asset_id}/content` 会读取恢复后的本地内容
  - applied `media_asset_content/recover_content` audit 会落账

还没做：

- QQ/Telegram 私有源凭证处理
- 平台 session / cookie / token 依赖下的重拉
- 自动后台重试
- 恢复后自动触发 Python AI enrichment

所以现在：

- 普通 HTTP/HTTPS 内容恢复不只是“代码已有”，而是 Go 已有 repo-owned
  isolated live smoke；
- 当前这台机器上真正活跃的边界仍是 operator content roots，而不是
  provider-specific private fetch；
- 平台私有源恢复还没有统一完成；
- dashboard 这块的 read-model gap 已收口，并已有 repo-owned current-turn
  live parity verifier。

## 三、配套能力还没补全到目标形态

### 1. MQ adapter 扩展

当前状态：

- NATS JetStream 是当前推荐实现
- Redis Streams / RabbitMQ 只有边界，没有实现
- 本轮已通过 `scripts/verify-mq-adapter-boundary.ps1` 和 dashboard
  `/api/dashboard/runtime-overview` 证明：selected provider、recommended
  external MQ、planned-only Redis/Rabbit 边界，以及 `queue_backend`
  capability matrix 都已在 live runtime 和 dashboard 中可见

这不算当前主线 blocker，但如果未来部署环境要求其它 MQ，就还没改。
当前剩余 gap 已收敛为“未来是否真的需要实现 Redis/Rabbit adapter”，而不是
read-model 或可见性不足。

### 2. Dashboard 仍可能有未来 drilldown 收尾工作

当前状态：

- 已有不少 detail 做成只读表格
- `media_asset_content_recovery` runtime-overview read-model gap 已收口
- `media_asset_content_recovery` dashboard/runtime live parity verifier 也已收口
- `queue_backend` capability matrix read-model gap 也已收口
- `queue_backend` 的 runtime-overview panel drilldown 也已收口
- `receiver_statuses` 的 runtime-overview panel drilldown 也已收口
- `receiver_leases` 的 runtime-overview panel drilldown 也已收口
- `knowledge_pipelines` 的 runtime-overview panel drilldown 也已收口
- `scheduler_jobs` 的 runtime-overview panel drilldown 也已收口
- `agent_workers` 的 runtime-overview panel drilldown 也已收口
- `runtime_workers` 的 runtime-overview panel drilldown 也已收口
- `agent_job_worker_coverage` 的 runtime-overview panel drilldown 也已收口
- `agent_job_metrics` 的 runtime-overview panel drilldown 也已收口
- `agent_job_pressure` 的 runtime-overview panel drilldown 也已收口
- `inbox_metrics` 的 runtime-overview panel drilldown 也已收口
- `inbound_dedupe_metrics` 的 runtime-overview panel drilldown 也已收口
- `observe_targets` 的 runtime-overview panel drilldown 也已收口
- `observe_capture` 的 runtime-overview panel drilldown 也已收口
- `delivery_adapters` 的 runtime-overview panel drilldown 也已收口
- `send_ledger_metrics` 的 runtime-overview panel drilldown 也已收口
- `outbox_metrics` 的 runtime-overview panel drilldown 也已收口
- `outbox_pressure` 的 runtime-overview panel drilldown 也已收口
- `external_lease_diagnostics` 的 runtime-overview panel drilldown 也已收口
- `agent_job_external_lease_readiness` / `agent_job_external_lease_plan`
  的 runtime-overview drilldown 也已收口
- `delivery_smoke_readiness` 的 runtime-overview drilldown 也已收口
- `agent_job_capacity_plan` / `agent_job_priority_plan` /
  `knowledge_job_planner_cutover_plan` / `outbound_cutover_plan`
  的 runtime-overview drilldown 也已收口
- `rag_eval_failures` 的 runtime-overview panel drilldown 也已收口
- `queue_topology` 的 repo-owned live verifier 和 unified verifier 证据也已收口
- `worker_control_executor` 的 repo-owned live verifier 和 unified verifier
  证据也已收口
- `qq_cutover_route_matrix` 的 runtime-overview dashboard drilldown 也已收口
- `control_audit` / `control_mutation_policy` 也已有 repo-owned dashboard
  live verifier，不再只是“前端面板字符串还在”
- `knowledge_pipelines` 现在也已有 repo-owned dashboard knowledge/RAG state
  boundary verifier，不再只是“panel 上有结构化表格”

当前结论：

- 当前这批已跟踪的 runtime-overview drilldown 已全部具备结构化只读展示。
- `queue_topology` 也已有 repo-owned live verifier，不再只是 dashboard 可见。
- `worker control executors` 也已有 repo-owned live verifier，不再只是
  靠静态残留分类描述。
- `qq_cutover_route_matrix` 也已同时具备 repo-owned live verifier 和 dashboard
  structured read-model，不再是 visibility gap。
- `control_audit` / `control_mutation_policy` 也已有 dashboard live parity
  verifier，不再是 unified goal 里最薄弱的 dashboard 证据点。
- `knowledge_pipelines` 现在也已有 repo-owned dashboard live parity verifier，
  当前 knowledge/RAG 剩余 gap 已收敛到 checkpoint snapshot 与外部 index
  metadata 的边界，而不是 dashboard 可见性。
- unified goal verifier 现在也会默认回写 `.codex-goal-verifier.json` 并附带
  顶层 `current_state` 摘要，因此当前剩余问题不再是“live 证据存在但 repo-local
  artifact 还是旧快照”，而是实际 cutover / executor / 凭证 blocker 本身。
- unified goal verifier 现在还会输出顶层 `migration_residuals`，把 QQ、
  Telegram、external lease result-ack、scheduler、worker control executors、
  media recovery private-source executor、dashboard read-model cleanup 和 MQ
  adapter boundary 统一固化成 machine-readable 当前 turn 分类。因此当前剩余
  gap 也不再是“残留口径每轮都要人工重写”，而是这些 blocker 何时被真实消除。
- unified goal verifier 现在还会输出顶层 `migration_bucket_summary`，直接把
  上述残留汇总进四个精确迁移桶：
  - `already_in_go_only_missing_live_verification`
  - `go_control_plane_present_but_real_executor_missing`
  - `still_not_fully_cut_over`
  - `explicitly_python_owned`
  这意味着当前剩余问题也不再是“final 里要再手工翻译一遍 bucket”，而是这四类
  blocker 本身何时被真实消除。
- 后续如果再出现新的 drilldown gap，属于新增 read-model 收尾，不是当前主线
  runtime-overview 仍停留在 raw JSON。
- unified goal verifier 现已同时提供 `dashboard_read_models.*` 与
  `checks.dashboard_read_models.*`；当前剩余不再是 dashboard evidence 的
  artifact 形状问题，而是实际 cutover / executor / credential blocker 本身。
- unified goal verifier 现在还会对 `dashboard_knowledge_rag_state_boundary`
  做一次短暂重试；当前剩余不再是 knowledge/RAG dashboard parity 已通过但
  `.codex-goal-verifier.json` 偶发误写成 partial fallback 的采样稳定性问题。
- unified goal verifier 现在还支持 `-StdoutMode summary|full|none`；
  当前剩余也不再是“artifact 已刷新但交互式调用仍要吃完整大 JSON stdout”
  这个 runbook 薄弱点，而是实际 cutover / executor / credential blocker 本身。

## 四、这些不是“没改完”，而是明确保留在 Python

后面讨论 Go 迁移时，下面这些不要再算进“还没迁移”的口径里。

### 1. 模型与推理相关

- LLM/provider 适配
- prompt / context 组装
- tool execution
- reasoning

### 2. 多模态与检索策略

- OCR / VLM
- image generation execution
- embeddings / rerank
- RAG 策略实验

### 3. 群知识与算法侧逻辑

- memory 抽取算法
- 群知识蒸馏
- evaluation scripts
- provider-specific fallback

这些按照当前架构约束，本来就应该留在 Python。

## 五、如果只看“现在最关键还没改完的 5 件事”

按优先级，当前最关键的未完成项是：

1. 重新登录或替换当前 NapCat / QQ rich-media 会话后，再让 QQ 图片真实 smoke 通过
2. 重新登录或替换当前 NapCat / QQ rich-media 会话后，再让 QQ 文件真实 smoke 通过
3. 先决定是否要实现 per-account/per-conversation-type/per-kind gate，以便把第一账号 private file 纳入 Go owner 而不误放开 first-account group file
4. 在 image 原生问题恢复后，再决定是否把当前 gate 扩大到全局 `image/file`
5. Telegram bot token 配置后做 backend smoke
6. operator-approved executor 类控制面落地

## 六、当前一句话结论

现在不是“Go 迁移还没开始”，也不是“只做了文档”。

准确状态是：

- **确定性 control plane 大部分已经迁到 Go**
- **QQ 私聊文本、群文本、text-only execution owner、WebSocket media staging 都已经证明可行**
- **系统级全量 cutover 还差 rich-media 平台 blocker 解除**
- **Telegram 后台仍卡在 bot token**

补充：receiver-status stale cleanup 已有 Go mutation 与 repo-owned live smoke；
这条线当前剩余不再是能力缺失，而是是否要在长期运行 runtime 上主动清理
`heartbeat_stale` receiver-status 记录。

补充：`knowledge_pipelines` dashboard read-model 目前不再存在 top-level parity
缺口；这条已回到 live verified dashboard state。
