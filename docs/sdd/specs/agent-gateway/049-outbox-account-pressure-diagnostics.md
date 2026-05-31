# 049 Outbox Account Pressure Diagnostics

Date: 2026-05-31

## 背景

Go runtime 已经持有 outbox delivery 的权威状态、重试、dead-letter 和 metrics。
但目前 metrics 主要按状态和 channel kind 汇总，不能直接回答某个账号是否积压过多
queued / dispatching delivery。后续如果要把平台发送限流、账号级 backpressure 和
worker 调度完全沉淀到 Go，需要先有稳定的只读压力诊断。

本切片先做 Go-owned outbox account pressure diagnostics：按
`channel_kind + account_id` 聚合 queued / dispatching / active / dead-letter，
并把 summary 接入 runtime overview。它不阻断真实发送，也不改变 outbox lease 或 retry
语义。

## Go / Python 边界

Go 负责：

- 从 Go-owned outbox state 计算账号级发送压力。
- 在 `/v1/outbox-metrics` 暴露 pressure 诊断。
- 在 `/v1/runtime-overview` summary/card 中暴露高压账号数、最大 active、最大 queued。

Python 负责：

- 不参与本切片。
- 继续通过 Go outbox / 兼容发送路径提交或执行平台发送，AI 决策和内容生成仍归 Python。

## 范围

- `OutboxMetricsView` 增加 `pressure` 字段：
  - `accounts`
  - `high_pressure_accounts`
  - `max_active`
  - `max_queued`
  - `by_account`
- account pressure key 为 `channel_kind:account_id`。
- active = queued + dispatching。
- high pressure 初始只读阈值：
  - queued >= 10
  - dispatching >= 5
  - active >= 10
- runtime overview 增加 `outbox_pressure` card。

## 不做

- 不做真正限流。
- 不暂停 worker，不拒绝 enqueue，不改变 retry。
- 不新增外部配置项；阈值先作为诊断常量，后续限流切片再设计 policy/env。
- 不让 Python 根据该指标改变 prompt 或发送策略。

## 验收

- Go outbox metrics service test 覆盖账号 pressure 聚合与 high pressure 判断。
- Go runtime overview service test 覆盖 summary 和 `outbox_pressure` card。
- `go test ./app/service -run "TestOutboxMetricsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics"` 通过。
- `go test ./...` 通过。
