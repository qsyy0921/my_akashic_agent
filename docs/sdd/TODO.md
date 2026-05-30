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
- [x] 配置并验证本机开发镜像源：Go 使用 `GOPROXY=https://goproxy.cn,direct`，Docker Desktop 用户级配置加入 `docker.m.daocloud.io` / `docker.1ms.run` registry mirror，并通过镜像域名直拉 `nats:2-alpine` 验证可用。
- [x] 完成 `agent_job` 外部租约评估并固化 Go 诊断门禁：`external_lease` 目前只允许 `outbox_delivery`，`agent_job` 继续走 Go state-store lease，直到补齐精确 job_id lease token、Python worker 心跳、幂等结果回写和 ack-after-result 协议。
- [x] 实现 `agent_job` result-ack 第一阶段：Go 生成并持久化 `lease_token`，HTTP 返回给 Python worker；running/succeeded/failed 支持 token fencing，Python image/knowledge/rag_eval worker 自动回传 token，旧 worker 不带 token 仍保持兼容。
- [x] 实现 `agent_job` heartbeat / lease renew：Go 增加 `/v1/jobs/{job_id}/renew` 和 `renewed` 生命周期事件，Python image/knowledge/rag_eval worker 在长任务执行期间后台续租，续租必须携带当前 `lease_token`。
- [x] 增加 `agent_job` 严格 token 模式：`AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true` 时 running/succeeded/failed 必须携带当前 `lease_token`，默认保持兼容模式，作为后续 NATS external lease cutover gate。
- [x] 增加 `agent_job` queue work id 精确租约入口：`POST /v1/jobs/lease-work` 按 `work_id`/`aggregate_id` 精确租约指定 job，不再依赖 `lease-next` 按类型抢任务，为 NATS work notification 驱动作准备。
- [x] 增加 `agent_job` 过期租约恢复入口：`POST /v1/jobs/recover-expired` 扫描 leased/running 且 lease 已过期的 job，未耗尽尝试次数则回到 pending，耗尽则 dead-letter，并写入 `lease_expired` 生命周期事件。

## 下一步

- [ ] 配置当前运行态 `AKASHIC_ONEBOT_WS_URLS="qq=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002"` 与 `AKASHIC_ONEBOT_ACCESS_TOKENS`，重启 `agent-runtime` 后确认 adapter enabled 日志。
- [ ] 做 QQ/NapCat Go adapter live send smoke：覆盖 1049511700/2365524513 双账号私聊文本、群文本、图片、文件；通过后再把对应 QQ channel alias 加入 `integrations.agent_runtime.outbound_channels`，并确认 recent-send / bot protocol 防循环仍生效。
- [ ] 补齐 `agent_job` NATS external lease 的剩余 result-ack 能力：ack/nack/term 映射、duplicate-delivery contract smoke，并把 recover-expired 接入安全的后台/启动恢复策略。

## 边界约束

- Go 负责确定性基础设施：路由状态、幂等、持久化存储、生命周期、重试、租约、checkpoint、资产和审计。
- Python 负责 AI 行为：模型调用、prompt、RAGFlow 上传、OCR/VLM、抽取启发式、排序、生成和快速实验。
- 不为架构形式过度拆分服务；优先在 `services/agent-runtime` 内复用现有 DDD/六边形分层，只有当职责和部署边界真正独立时才新增服务。
- 每完成一个迁移切片，都必须更新本中文 TODO，以及对应 SDD spec/review 文档。
