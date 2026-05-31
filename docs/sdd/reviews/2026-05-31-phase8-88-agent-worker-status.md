# Phase 8.88 Review: Go-owned Agent Worker Status

日期：2026-05-31

## 结论

通过。这个切片把 Python AI worker 的 liveness 与任务执行状态登记迁移到 Go `agent-runtime`，属于确定性运行态和诊断控制面，符合“Go 做基础设施、Python 做 AI pipeline”的边界。

## 设计审查

- `domain/model` 新增 `AgentWorkerStatus`，状态枚举和 stale heartbeat 判断在领域层内聚。
- `app/service` 负责编排 report/list、持久化和 read-time stale 降级。
- `infrastructure/agentworkerstatusstore` 使用 JSON 文件态，遵循现有 runtime state 默认持久化策略。
- `trigger/http` 暴露 report/list，不触发平台发送、模型调用或 worker 调度。
- Python worker 上报是 best-effort，失败只记 warning，不影响 image/knowledge/rag_eval/outbox 任务执行。

## 风险

- 现在 worker status 只做诊断，不参与调度决策；后续如果要用于自动恢复，必须先补租约和 fencing。
- outbox worker 仍可能由 Go local worker 或 Python compatibility worker 二选一执行；status 只描述 Python worker 本身，不等价于投递系统整体健康。

## 验收门

- Go 单测：service、store、HTTP handler。
- Python 单测：client payload 和 image worker idle 上报。
- Runtime overview 应展示 `Agent Workers` 卡片，且 detail 中包含 `agent_workers`。
