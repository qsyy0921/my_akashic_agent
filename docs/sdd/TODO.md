# Akashic Go 迁移 TODO

最后更新：2026-05-30

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

## 下一步

- [ ] 将普通 `message_push` / `OutboundPort` 发送路径接入 Go `/v1/outbound` + outbox worker，使 Telegram Go `DeliveryAdapter` 真正覆盖常规回复和主动推送路径。
- [ ] 评估并实现 QQ/NapCat Go `DeliveryAdapter`：先明确 OneBot HTTP/WebSocket 发送边界、双账号路由、二维码登录状态和防循环交互策略。
- [ ] 增加 Go/Python contract fixtures，覆盖 checkpoint、inbox replay、outbox delivery、media asset content、job event stream。
- [ ] 将 RAG evaluation jobs 做成 Go-owned 生命周期记录，Python 作为 eval worker。
- [ ] 增加运行态 dashboard 面板，展示 runtime health、worker leases、stale jobs、dead letters、checkpoint lag、job event stream。
- [ ] 评估是否把 outbox delivery state 也接入同一类 lifecycle event stream，再引入外部队列后端。

## 边界约束

- Go 负责确定性基础设施：路由状态、幂等、持久化存储、生命周期、重试、租约、checkpoint、资产和审计。
- Python 负责 AI 行为：模型调用、prompt、RAGFlow 上传、OCR/VLM、抽取启发式、排序、生成和快速实验。
- 每完成一个迁移切片，都必须更新本中文 TODO，以及对应 SDD spec/review 文档。
