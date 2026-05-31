# 055 Queue Execution Owner Diagnostics

Date: 2026-05-31

## 背景

Akashic runtime 目前已经存在多条执行路径：

- outbox delivery 可由 Go local state-store worker 执行；
- outbox delivery 未来可由 NATS `external_lease` executor 执行；
- generic agent_job 的 AI 执行仍由 Python AI workers 承担；
- NATS `external_lease` 对 agent_job 只做 result-ack，不执行模型/RAG/Memory。

这些边界如果只靠 mode、source 和 gate 推断，前端/运维/后续开发很容易误以为
`external_lease` 已经让 Go 执行了 agent_job AI 任务。本切片把执行所有权做成
显式只读诊断。

## Go / Python 边界

Go 负责：

- 暴露 outbox delivery 当前执行 owner。
- 暴露 agent_job 当前执行 owner。
- 在 runtime overview summary 中同步这些 owner 字段。
- 保持 external lease gate 与 local worker 冲突检查不变。

Python 负责：

- 继续执行 agent_job 的 AI work：模型/provider 路由、Prompt 和上下文构造、工具调用编排、
  Memory/RAG 抽取与合并、chunking、retrieval/ranking、Embedding/Rerank、OCR/VLM、
  图片生成执行、群知识沉淀、离线评估和 provider fallback。
- 作为 worker 通过 Go domain API 获取 lease、续租、回写结果并上报 worker status；
  不拥有队列生命周期、幂等、防循环、资产访问控制或审计权威状态。
- 在 agent_job result 写回 Go 后，由 Go/NATS result-ack 路径确认 queue 通知。

## 范围

- `QueueBackendView` 增加：
  - `outbox_execution_owner`
  - `agent_job_execution_owner`
- owner 规则：
  - outbox:
    - `nats_external_lease`：external lease gate ready；
    - `go_local_outbox_worker`：local outbox worker enabled；
    - `go_state_store_api`：默认只由 Go state-store API/手动 lease 暴露。
  - agent_job:
    - `python_ai_worker_with_nats_result_ack`：agent_job NATS result-ack scope ready；
    - `python_ai_worker_state_store_lease`：默认 Python 通过 Go state-store lease API 执行。
- runtime overview summary 增加对应字段。
- 测试覆盖 local 默认、local worker、external lease、agent_job result-ack。

## 不做

- 不改变执行行为。
- 不启用 external lease cutover。
- 不把 agent_job AI 执行迁到 Go。
- 不修改 Python worker。

## 验收

- `/v1/queue-backend` 能直接读到 execution owner。
- `/v1/runtime-overview.summary` 同步 owner 字段。
- local worker 与 external lease 配置下 owner 判断正确。
- agent_job result-ack ready 时 owner 仍明确为 Python AI worker。
- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*(DefaultsLocal|ExternalLease|OutboxWorker|AgentJobResultAck)" -count=1 -v` 通过。
- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v` 通过。
- `go test ./...` 通过。
