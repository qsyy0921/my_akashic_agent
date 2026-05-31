# Akashic Go 迁移 TODO

最后更新：2026-05-31

## 已完成

- [x] 将 Go 服务边界重命名为 `services/agent-runtime`。
- [x] Go 代码统一放在 `services/agent-runtime`，并保持 DDD + 六边形架构分层。
- [x] 增加 Go 侧 shadow audit 入站记录与查询 API。
- [x] 增加 Go 侧媒体资产注册表和安全内容访问。
- [x] 通过 Go media asset 路由为 dashboard 提供附件链接。
- [x] 增加 Go 侧 send ledger，用于 recent-send 和 echo-loop 防护。
- [x] 增加 Go 侧 outbox 生命周期、持久化、租约、重试和失败分类。
- [x] 增加 Go 侧通用 `AgentJob` 生命周期和持久化。
- [x] 将 observe-only 群记忆抽取生命周期迁移到 Go generic jobs，Python 继续作为抽取 worker。
- [x] 将 group-memory 和 RAGFlow 的源消息回放迁移到 Go inbox API。
- [x] 增加 Go 侧 knowledge checkpoints，用于 RAGFlow `rag_ingest` 游标。
- [x] 修复 Go runtime JSON 响应的 UTF-8 声明，避免 PowerShell 中文乱码。
- [x] 增加 Go-owned knowledge checkpoints 的 dashboard 可见性。
- [x] 修复 Windows 本地启动时 Go media content 安全根目录发现问题，让 QQ 图片/文件附件可从 dashboard 打开。
- [x] 增加 Go-owned memory/RAG worker 诊断接口，聚合 generic job 与 checkpoint 状态。
- [x] 在 generic job leasing 后增加 Go-owned durable stream，使用 JSONL 记录 job 生命周期事件。
- [x] 修复 dashboard 媒体附件缩略图回退逻辑，前端可根据 `asset_id` 直接生成 Go media content 代理链接。
- [x] 增加 Go-owned proactive scheduling state API 和 JSON 持久化，覆盖 delivery 去重、窗口计数、context-only 节流和 drift 间隔标记。
- [x] 将 Python proactive loop 接入 Go proactive scheduling state，并保留 SQLite fallback。
- [x] 修复 dashboard 旧附件链接兼容问题：Go media content 404/403 时，仅对 workspace uploads 内的文件名做安全回退。
- [x] 强化 dashboard 媒体附件兜底：Go media content 403/404 时，可根据 runtime asset 元数据回退到 workspace uploads 内的同名本地镜像，避免 QQ 图片/文件在前端显示为 broken image。
- [x] 增加 Go-owned DeliveryAdapter dispatch plan：Go 负责 outbox 路由解析、附件拆分和 file URI 规范化，Python 兼容 worker 只执行平台发送。
- [x] 增加 Go-owned Telegram DeliveryAdapter：Go 通过 Telegram Bot API 执行 outbox text/photo/document 发送，Python worker 对 Telegram 优先走 Go，adapter 不可用时回退旧发送链路。
- [x] 将普通 `message_push` / `OutboundPort` 发送路径接入 Go `/v1/outbound` + outbox worker；当前仅 Telegram 进入 Go outbound，QQ/NapCat 继续保留 Python direct fallback，避免未完成适配器导致双发或漏发。
- [x] 增加 Go-owned QQ/NapCat OneBot HTTP `DeliveryAdapter`：Go 可按 `qq_1049511700` / `qq_2365524513` 等 channel alias 执行私聊/群聊文本、图片、文件发送；默认仍需显式配置 endpoint 后才启用，避免影响现有观察群链路。
- [x] 确认当前两个 NapCat 容器只启用了 OneBot WebSocket server，普通 HTTP 请求返回 `426 Upgrade Required` 是预期现象；Go OneBot adapter 已扩展支持 WebSocket action，并用只读 `get_login_info` 验证 3001=1049511700、3002=2365524513 在线。
- [x] 将 Python outbox worker 的 Go dispatch 尝试范围改为读取 `integrations.agent_runtime.outbound_channels`，QQ channel alias 只有显式加入该列表后才会走 Go OneBot adapter。
- [x] 增加 Go/Python contract fixtures，覆盖 checkpoint、inbox replay、outbox delivery、media asset content、job event stream，并用两端测试校验关键边界字段。
- [x] 基于新增 contract fixtures 增加 dashboard/worker contract smoke：checkpoint dashboard 可计算 lag，media content 经 dashboard 代理返回，job event stream 可经 dashboard API 读取，knowledge worker 使用真实 checkpoint cursor。
- [x] 将 RAG evaluation jobs 做成 Go-owned 生命周期记录，Python 作为 opt-in eval worker 执行离线 group-memory/RAG fixture 并回写指标。
- [x] 增加运行态 dashboard 总览面板，聚合 runtime health、worker leases、stale jobs、dead letters、checkpoint lag、job event stream 和 `rag_eval` 失败摘要。
- [x] 为 dashboard 增加 `rag_eval` 结果趋势与质量门摘要面板，区分质量门失败和基础设施失败，并展示 per-question 评测结果。
- [x] 将 outbox delivery state 接入 Go-owned durable lifecycle event stream，新增 `/v1/outbox-events`、JSONL 持久化和 runtime overview 聚合。
- [x] 评估并设计外部队列后端：确定 NATS JetStream 作为第一实现目标，支持后续多 goroutine 并发消费，保留 Redis Streams/RabbitMQ 适配边界，并新增 `/v1/queue-backend` 只读迁移诊断。
- [x] 实现 NATS JetStream `shadow_publish` 适配器：创建 outbox/generic job 时先写 Go state store，再发布 work notification；当前不执行外部租约。
- [x] 增加 NATS JetStream `shadow_publish` 诊断对账：记录发布成功/失败、按 subject 的通知数，并在 `/v1/queue-backend` 中对比 Go state store 与 event stream 的样本差异。
- [x] 实现 NATS JetStream `dual_read_compare`：Go 以多 goroutine 消费 MQ work notification，并只读校验 queue candidate 与 Go outbox/job 权威状态，不执行发送或 Python worker 副作用。
- [x] 增加 NATS JetStream `external_lease` 切换门诊断：明确 ack/nack、retry、dead-letter、rollback 策略和必过条件；当前保持阻断，不新增服务、不执行真实外部租约。
- [x] 完成本地 NATS JetStream live smoke：在 `dual_read_compare` 模式下验证 `shadow_publish` 成功 1 次、compare 匹配 1 次、mismatch 为 0，并确认 `AKASHIC_QUEUE_CONSUMER_CONCURRENCY=2` 生效。
- [x] 实现最小化 NATS JetStream `external_lease` outbox 执行器：Go 按 queue work id 租约 outbox、调用 Go DeliveryAdapter dispatch、按 Go 生命周期结果执行 ack/nack/term；默认仍需显式 cutover 与 smoke flag，不新增独立服务。
- [x] 完成 `external_lease` 本地 smoke：使用 NATS + fake adapter 验证 outbox 成功 ack、可重试失败延迟 nack、终态失败 ack、unsupported work term；不触发真实 QQ/Telegram 发送。
- [x] 配置并验证本机开发镜像源：Go 使用 `GOPROXY=https://goproxy.cn,direct` 和 `GOSUMDB=sum.golang.google.cn`，Docker Desktop 用户级配置加入 `docker.m.daocloud.io` / `docker.1ms.run` registry mirror，并通过镜像域名直拉 `nats:2-alpine` 验证可用。
- [x] 完成 `agent_job` 外部租约评估并固化 Go 诊断门禁：`external_lease` 目前只允许 `outbox_delivery`，`agent_job` 继续走 Go state-store lease，直到补齐精确 job_id lease token、Python worker 心跳、幂等结果回写和 ack-after-result 协议。
- [x] 实现 `agent_job` result-ack 第一阶段：Go 生成并持久化 `lease_token`，HTTP 返回给 Python worker；running/succeeded/failed 支持 token fencing，Python image/knowledge/rag_eval worker 自动回传 token，旧 worker 不带 token 仍保持兼容。
- [x] 实现 `agent_job` heartbeat / lease renew：Go 增加 `/v1/jobs/{job_id}/renew` 和 `renewed` 生命周期事件，Python image/knowledge/rag_eval worker 在长任务执行期间后台续租，续租必须携带当前 `lease_token`。
- [x] 增加 `agent_job` 严格 token 模式：`AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true` 时 running/succeeded/failed 必须携带当前 `lease_token`，默认保持兼容模式，作为后续 NATS external lease cutover gate。
- [x] 增加 `agent_job` queue work id 精确租约入口：`POST /v1/jobs/lease-work` 按 `work_id`/`aggregate_id` 精确租约指定 job，不再依赖 `lease-next` 按类型抢任务，为 NATS work notification 驱动作准备。
- [x] 增加 `agent_job` 过期租约恢复入口：`POST /v1/jobs/recover-expired` 扫描 leased/running 且 lease 已过期的 job，未耗尽尝试次数则回到 pending，耗尽则 dead-letter，并写入 `lease_expired` 生命周期事件。
- [x] 增加 `agent_job` external lease result-ack 映射：Go 不抢占执行 Python 任务，只按 AgentJob 权威状态对队列通知做 ack/nack/term；pending/running 延迟 nack，terminal ack，missing term，failed 自动 retry 后 nack，expired lease 先恢复再决定 nack/ack，并覆盖重复终态投递单元 smoke。
- [x] 增加 `agent_job` NATS subject 显式扩容门禁：默认 external lease consumer 仍只订阅 outbox；只有 `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED=true`、`AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED=true`、`AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED=true`、`AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true` 和基础 cutover 门禁全部满足时，才扩展为 `outbox_delivery_and_agent_job_result_ack`。
- [x] 完成 `agent_job` NATS 级 duplicate terminal delivery smoke：临时启动本地 NATS JetStream 容器，发布同一 terminal AgentJob 的两条不同 queue notification，验证两条都 ack 且不会触发 Python/平台副作用。
- [x] 增加可选 Go runtime `agent_job` 过期租约后台恢复 runner：默认关闭；开启 `AKASHIC_AGENT_JOB_RECOVERY_ENABLED=true` 后定时触发同一 `RecoverExpiredLeases` 用例，支持 interval、limit、run-on-start 配置，并保持领域规则只在 domain/app 层。
- [x] 完成 `agent_job` NATS result-ack 状态流 dry-run：临时本地 NATS JetStream 中验证同一 job 的 pending 通知 delayed nack、running 通知 delayed nack、Python-style succeeded 写回后 terminal ack；并新增 `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED=true` 作为 live subject 扩容门禁。
- [x] 增加 Go-owned delivery adapter 只读诊断接口 `GET /v1/delivery-adapters`：可查看 Telegram/OneBot channel alias、transport、endpoint 是否配置、token 是否配置和脱敏 endpoint，用于替代只看日志确认 adapter enabled，且不会触发真实平台发送。
- [x] 增加 Go-owned 私聊 echo 只读判断接口 `GET /v1/send-ledger/private-echo`：Go 统一处理文本回流以及空文本图片/文件/转发 marker 回流判断，Python QQ channel 优先调用该接口并保留旧 `recently_sent` fallback。
- [x] 将 Go-owned delivery adapter 诊断接入 Python `AgentGatewayClient` 和 runtime overview dashboard：前端可直接看到 Telegram/OneBot adapter enabled/disabled 状态，不需要依赖日志确认。
- [x] 将 Go-owned queue backend 诊断接入 Python `AgentGatewayClient` 和 runtime overview dashboard：前端可查看 NATS/本地队列 provider、mode、多 goroutine consumer concurrency、max-in-flight 与 external lease gate 阻断原因。
- [x] 增加 Go-owned delivery dispatch readiness 只读接口 `POST /v1/delivery-dispatch/readiness`：Go 统一判断 outbox 路由计划是否有可用 DeliveryAdapter，返回 missing channel 和 side_effect=none，Python client 可读取但不改变现有发送行为。
- [x] 将 Python outbox worker 的 Go dispatch 决策接入 Go readiness：配置为 Go dispatch 候选的 channel 会先由 Go 判断 adapter 是否 ready；若缺失 adapter，则直接执行 Go readiness 返回的 plan，不再先尝试必然失败的 runtime dispatch。
- [x] 将 outbox dashboard 详情接入 Go dispatch readiness：详情页只读展示 adapter ready/missing channel、side_effect=none 和 dispatch plan，便于 QQ/NapCat adapter cutover 前诊断缺失 channel，不触发真实发送。
- [x] 将 `agent_job` 过期租约恢复接入 Agent Jobs dashboard：前端可触发 Go `/v1/jobs/recover-expired`，展示扫描/恢复/死信结果，并只暴露 `lease_token_present` 避免泄漏 token 值。
- [x] 增加 Go-owned `agent_job` metrics endpoint `GET /v1/job-metrics`：Go 聚合 job 状态、类型分布、生命周期吞吐和 dead-letter 趋势，runtime overview dashboard 只读展示该 Go 指标口径。
- [x] 增加 Go-owned outbox metrics endpoint `GET /v1/outbox-metrics`：Go 聚合投递状态、channel 分布、生命周期吞吐和 dead-letter 趋势，runtime overview dashboard 只读展示该 Go 指标口径。
- [x] 增加 Go-owned inbox metrics endpoint `GET /v1/inbox-metrics`：Go 聚合原始观察消息、observe-only 占比、附件采集、会话 sender 和 seq cursor，用于 runtime overview 观察 QQ 群数据采集质量。
- [x] 增加 Go-owned send ledger metrics endpoint `GET /v1/send-ledger/metrics`：Go 聚合 recent-send 防循环记录、bot/conversation 分布和重复 content hash，用于 runtime overview 审计双账号互聊回流风险。
- [x] 增加 Go-owned runtime overview aggregate endpoint `GET /v1/runtime-overview`：Go 聚合 adapter、queue、send ledger、inbox、agent job、outbox 和 knowledge diagnostics，Python dashboard 优先读取该聚合口并保留旧多接口 fallback。
- [x] 增加可选 Go-owned local outbox delivery worker：默认关闭；开启 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 后由 Go state-store lease outbox、调用 Go DeliveryAdapter dispatch 并回写 succeeded/failed，减少 Python outbox worker 的确定性基础设施职责。
- [x] 增加 Go-owned runtime worker diagnostics endpoint `GET /v1/runtime-workers`：Go 汇总 agent job recovery、local outbox worker、NATS shadow/dual-read/external-lease worker 的 enabled/running/config 状态，并接入 runtime overview dashboard。
- [x] 增加 Go-owned delivery adapter live health endpoint `GET /v1/delivery-adapters/health`：Go 只读调用 OneBot `get_login_info` 与 Telegram `getMe`，返回 channel alias 是否 reachable/authenticated/account_id，side_effect 固定为 none，便于 QQ/NapCat live send 前验证连接。
- [x] 将 delivery adapter live health 接入 runtime overview dashboard：新增手动 `/api/dashboard/runtime-overview/delivery-adapter-health` 代理和 `Delivery Adapters` 详情页 Probe Health 按钮；不会在概览加载时自动 ping QQ/Telegram。
- [x] 增加 Go-owned runtime config 只读诊断 `GET /v1/runtime-config`：脱敏展示当前进程地址、bot ids、OneBot/Telegram env、预期 QQ channel alias、缺失项、worker/cutover 开关，并接入 `GET /v1/runtime-overview` 的 Runtime Config 卡片。
- [x] 配置并验证当前运行态 OneBot/Telegram adapter：重建并重启 `agent-runtime`，启用 `qq`、`qq_1049511700`、`qq_2365524513` WebSocket alias，确认 `/v1/runtime-config`、`/v1/delivery-adapters`、`/v1/delivery-adapters/health`、`/v1/runtime-workers`、`/v1/runtime-overview` 和 dashboard runtime overview 均可用；Go outbox delivery worker 仍保持关闭，未执行真实发送。
- [x] 强化 `GET /v1/runtime-config` 脱敏策略：token/secret 类环境变量只返回 `redacted` 或 `channel=redacted`，不再暴露首尾片段；`AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN` 作为布尔配置保留 true/false 可见性。
- [x] 增加 Go-owned delivery smoke readiness 矩阵 `POST /v1/delivery-smoke/readiness`：在未创建 outbox、未触发 QQ/Telegram 平台发送的情况下，复用 Go dispatch planner 检查双 QQ 私聊、可选群文本、合成图片和文件 case 是否有可用 DeliveryAdapter，并返回 `side_effect=none`、missing channel 与阻断原因。
- [x] 将 delivery smoke readiness 接入 Python `AgentGatewayClient` 和 runtime overview dashboard：新增手动 `/api/dashboard/runtime-overview/delivery-smoke-readiness` 代理与 `Delivery Adapters` 详情页 Smoke Readiness 操作，可输入群号并触发 Go 只读预检，仍不执行平台发送。
- [x] 增加 Go-owned observe target diagnostics：Python 启动时将 `config.toml` 中 observe-only QQ 群同步到 Go `/v1/observe-targets/sync`，Go 负责 source-bound 保存、校验、`GET /v1/observe-targets` 查询和 runtime overview `Observe Targets` 卡片；不改变当前 QQ 收消息和回复逻辑。
- [x] 重建并重启当前本地 Go `agent-runtime` 与 dashboard，确认 runtime overview 显示 6 个 observe-only QQ 群：`164369633`、`187890369`、`27234224`、`284331268`、`3219982`、`956393163`，`side_effect=none`。
- [x] 恢复当前本地 QQ 观察链路：Docker API 已恢复，两个 NapCat 容器在线；重启 Python 主服务后 `1049511700 -> ws://localhost:3001`、`2365524513 -> ws://localhost:3002` 均成功启动，Go delivery adapter health 显示 OneBot/Telegram 4 个 adapter 全部 authenticated，runtime overview 已记录 `3219982` 的 observe-only 群消息。
- [x] 增加 Go-owned receiver status diagnostics：Python QQ/Telegram 接收端启动、失败和 Telegram polling conflict 会上报到 Go `/v1/receiver-statuses/report`；Go 负责校验、聚合、`GET /v1/receiver-statuses` 和 runtime overview `Receiver Statuses` 卡片。当前 live smoke 显示两个 QQ receiver connected：`qq:1049511700:qq`、`qq:2365524513:qq_2365524513`，并正确记录 Telegram `status=suspended`、`reason=getupdates_conflict`。
- [x] 增加 Go-owned receiver lease control：Go 提供 `/v1/receiver-leases/acquire|renew|release` 和 `GET /v1/receiver-leases`，Telegram polling 启动前先获取租约、运行中续租、停止或 conflict 时释放；runtime overview/dashboard 展示 active/expired lease 计数，列表响应只暴露 `lease_token_present` 不泄漏 token 值。当前 live smoke 显示 `telegram:7689386159:telegram` 有 1 个 active lease，两个 QQ receiver 和 Telegram receiver 均为 connected。
- [x] 增加 Go-owned observe capture diagnostics：Go 提供 `GET /v1/observe-capture-diagnostics`，聚合 observe targets、receiver status、inbox events、media assets 和安全 media content probe，给出每个观察群的文本/图片/文件覆盖、content ready 数量和 blockers，并接入 runtime overview/dashboard。当前 live smoke 显示 6 个 QQ observe-only 群均由 `qq:1049511700:qq` 连接，但因重启后暂无新样本，文本/图片/文件覆盖均为待验证 warn。
- [x] 将 Go runtime 未显式配置的确定性状态默认改为文件持久化：`agent-runtime` 自动使用 `.akashic-workspace/agent-runtime` 保存 observe targets、AgentJob、job events、media assets、send ledger、outbox、outbox events、inbox、knowledge checkpoints 和 proactive state；`AKASHIC_RUNTIME_STATE_DIR` 可统一覆盖，`AKASHIC_RUNTIME_STATE_DIR=memory` 或单项 `*_DSN=memory` 可显式回到内存态，避免重启后 observe capture 样本和观察群配置丢失。

## 下一步

- [ ] 继续验证 QQ 群实时采集质量：让任一观察群产生一条文本、一张图片、一个文件，然后查看 Go `/v1/observe-capture-diagnostics` 是否从 warn 变为至少 text/image/file 对应 covered；重启 `agent-runtime` 后再次确认 covered 状态仍保留，同时确认 Python QQ channel、Go inbox metrics、media asset content、dashboard 附件预览链路都能完整记录，且 observe-only 不回复。
- [ ] 补 receiver status 的重启恢复策略：当前只重启 Go 后 observe targets/inbox/media 会持久保留，但 receiver connected 状态需要 Python 接收端再次上报；下一步应让 Python QQ/Telegram receiver 定时 heartbeat 到 Go，或在 Go 侧增加带 TTL 的 last-known 状态恢复，避免 `/v1/observe-capture-diagnostics` 在 Go 重启后误报 `receiver_not_connected`。
- [ ] 若 Telegram `getUpdates` conflict 再次出现，先看 Go `/v1/receiver-statuses` 是否显示 `status=suspended`、`reason=getupdates_conflict`，并确认 `/v1/receiver-leases` 是否没有重复 Akashic receiver；如果仍冲突，说明外部非 Akashic polling 进程占用 token，需要停止外部进程或改成 webhook。
- [ ] 做 QQ/NapCat Go adapter live send smoke：覆盖 1049511700/2365524513 双账号私聊文本、群文本、图片、文件；通过后再把对应 QQ channel alias 加入 `integrations.agent_runtime.outbound_channels`，或改由 `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 的 Go local outbox worker 接管，并确认 recent-send / bot protocol 防循环仍生效。
- [ ] 继续收敛 Go/Python 分工：检查是否还有确定性 runtime 状态、幂等、调度、资产、队列、审计逻辑仍散落在 Python，能迁移则按 SDD 切片迁移。

## 边界约束

- Go 负责确定性基础设施：路由状态、幂等、持久化存储、生命周期、重试、租约、checkpoint、资产和审计。
- Python 负责 AI 行为：模型调用、prompt、RAGFlow 上传、OCR/VLM、抽取启发式、排序、生成和快速实验。
- 不为架构形式过度拆分服务；优先在 `services/agent-runtime` 内复用现有 DDD/六边形分层，只有当职责和部署边界真正独立时才新增服务。
- 每完成一个迁移切片，都必须更新本中文 TODO，以及对应 SDD spec/review 文档。
