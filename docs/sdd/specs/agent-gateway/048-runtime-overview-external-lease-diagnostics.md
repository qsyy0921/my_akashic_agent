# 048 Runtime Overview External Lease Diagnostics

Date: 2026-05-31

## 背景

`047-external-lease-execution-diagnostics` 已经在 `/v1/queue-backend` 暴露
`external_lease.diagnostics`，可以看到 external queue lease 的 ack/nack/term
执行结果。但 runtime overview 是 dashboard 的稳定总览入口，目前只显示 queue backend
provider/mode/ready 状态，不能直接在 summary/card 层看到 external lease 消费质量。

本切片把 external lease diagnostics 汇总到 Go-owned runtime overview，方便前端和
live smoke 一眼判断 MQ 消费是否发生、是否有 executor error、是否产生大量 delayed nack。

## Go / Python 边界

Go 负责：

- 从 `QueueBackendView.ExternalLease.Diagnostics` 读取只读诊断快照。
- 在 runtime overview summary 中暴露执行总数、错误数、ack/nack/term 计数。
- 增加 runtime overview card，保留 queue backend 原始 detail，方便前端展开查看。

Python 负责：

- 不参与本切片。
- 继续作为 AI worker 执行 AgentJob；Go overview 只展示状态，不执行 AI 任务。

## 范围

- `runtimeOverviewSummary` 增加：
  - `queue_external_lease_executed_total`
  - `queue_external_lease_error_total`
  - `queue_external_lease_ack`
  - `queue_external_lease_nack`
  - `queue_external_lease_term`
- `runtimeOverviewCards` 增加 `external_lease_diagnostics` card。
- card 状态规则：
  - 无 diagnostics 或未启用：`muted`
  - `error_total > 0`：`danger`
  - `nack > 0` 或 `term > 0`：`warn`
  - `executed_total > 0` 且无异常：`ok`

## 不做

- 不新增 HTTP endpoint。
- 不改变 `/v1/queue-backend` 原始结构。
- 不触发 queue lease、ack、nack、term 或平台发送。
- 不让 Go 执行 Python AI worker。

## 验收

- Go runtime overview service test 覆盖 summary 字段和 card 状态。
- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics` 通过。
- `go test ./...` 通过。
