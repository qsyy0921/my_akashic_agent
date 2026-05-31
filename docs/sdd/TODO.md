# Akashic Go 迁移 TODO

最后更新：2026-05-31

## 使用规则

- 本文件只保留本轮迭代必须全部完成的短待办，原则上不超过 10 条。
- 已完成迁移记录放到 `docs/sdd/DONE.md`。
- 运行态观察、live smoke 和人工验证项放到 `docs/sdd/LIVE_CHECKS.md`。
- 长期规划、未来想法、非本轮任务放到 `docs/sdd/BACKLOG.md`，不要堆在本文件。
- 每轮结束前，本文件必须清空，或者只保留明确阻塞项并写明阻塞原因。
- 每完成一个迁移切片：更新本 TODO、补 SDD spec/review，并把完成项归档到 DONE。

## 本轮未完成

- 当前无未完成项。下一轮开始时，从 `docs/sdd/BACKLOG.md` 选择本轮可完成的任务，再写入本文件。

## 边界约束

- Go 负责确定性基础设施：路由状态、幂等、持久化存储、生命周期、重试、租约、checkpoint、资产和审计。
- Python 负责 AI 行为：模型调用、prompt、RAGFlow 上传、OCR/VLM、抽取启发式、排序、生成和快速实验。
- 不为架构形式过度拆分服务；优先在 `services/agent-runtime` 内复用现有 DDD/六边形分层，只有当职责和部署边界真正独立时才新增服务。
- 每完成一个迁移切片，都必须更新本中文 TODO，以及对应 SDD spec/review 文档。
