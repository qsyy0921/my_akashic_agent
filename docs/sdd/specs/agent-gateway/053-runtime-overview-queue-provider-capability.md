# 053 Runtime Overview Queue Provider Capability

Date: 2026-05-31

## 背景

`052-queue-provider-capability-diagnostics.md` 已经让 `/v1/queue-backend`
暴露 MQ provider capability matrix。但 dashboard 和监控多数场景读取的是
`/v1/runtime-overview` 的 summary/card。如果 provider 能力只藏在 queue backend
detail 里，前端仍需要重复解析复杂结构。

本切片把当前选中 provider 的关键能力提升到 runtime overview summary，并让
`queue_backend` card 显示 provider 的推荐阶段。它是只读诊断增强，不改变
NATS cutover gate、不触发真实平台发送、不修改 Python AI worker。

## Go / Python 边界

Go 负责：

- 从 `QueueBackendView.SelectedProviderCapability` 汇总稳定字段。
- 在 runtime overview summary 暴露当前 MQ provider 的实现状态、推荐状态、
  并发消费、delayed nack、external lease 和 agent_job result-ack 能力。
- 保持 queue backend card detail 仍包含完整 `QueueBackendView`。

Python 负责：

- 不参与 MQ provider capability 计算。
- 继续作为 AI worker 执行模型、Prompt、Memory/RAG、OCR/VLM 和图片生成。

## 范围

- `runtimeOverviewSummary` 增加：
  - `queue_provider_status`
  - `queue_provider_recommended`
  - `queue_provider_recommended_phase`
  - `queue_provider_implemented`
  - `queue_provider_supports_concurrent_consumers`
  - `queue_provider_supports_delayed_nack`
  - `queue_provider_supports_external_lease`
  - `queue_provider_supports_agent_job_result_ack`
- `queue_backend` card value 在有 recommended phase 时显示
  `provider/mode (phase)`。
- 测试覆盖 NATS provider capability 被 runtime overview 汇总。

## 不做

- 不新增 provider adapter。
- 不改变 NATS external lease 或 agent_job result-ack gate。
- 不修改 Python dashboard API fallback。

## 验收

- runtime overview summary 可直接读取 MQ provider capability 摘要。
- queue backend card value 带推荐阶段。
- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v` 通过。
- `go test ./...` 通过。
