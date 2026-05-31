# 051 External Lease Outbox Account Rate Limit

Date: 2026-05-31

## 背景

`050-outbox-account-rate-limit.md` 已经让 Go local outbox delivery worker
支持账号级节流，但 NATS `external_lease` outbox executor 是另一个 Go 侧真实发送
执行路径。未来一旦 external lease cutover，它不能绕过同一套账号节流策略。

本切片补齐 external lease outbox executor 的账号级节流，并把 local worker 中的
限流状态抽成复用的 Go runtime policy。它不启用 external lease cutover，不改变
Python AI worker，也不改变未配置节流时的发送行为。

## Go / Python 边界

Go 负责：

- 维护可复用的 `OutboxAccountRateLimiter`。
- local outbox worker 和 external lease outbox executor 使用同一类 policy。
- external lease executor 在租约前检查账号节流；被节流时返回 `nack` +
  `delivery_rate_limited`，不拿 Go outbox lease，不调用 delivery adapter。
- runtime worker diagnostics 暴露 external lease 是否配置账号节流。

Python 负责：

- 不参与本切片。
- 继续执行模型、Prompt、Memory/RAG、OCR/VLM、图片生成和 provider fallback。

## 范围

- 新增共享 rate limiter：基于 `channel_kind:account_id`，支持 min interval 和
  window max。
- local worker 使用共享 limiter 替代私有实现。
- external lease outbox executor 使用共享 limiter。
- external lease 被节流时：
  - `Disposition = nack`
  - `Reason = delivery_rate_limited`
  - 不调用 `OutboxService.Lease`
  - 不调用 `DeliveryDispatchService.Dispatch`
- 使用上一切片的 env：
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MIN_INTERVAL_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_WINDOW_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MAX_PER_WINDOW`

## 不做

- 不做持久化/分布式 rate limit state。
- 不改变 NATS consumer ack/nack delay 策略。
- 不打开 external lease cutover gate。
- 不把平台风控策略写入 Python prompt。

## 验收

- domain/app-level shared limiter tests 覆盖 min interval、window max、prune。
- local outbox worker rate-limit test 继续通过。
- external lease service test 覆盖被节流时 nack，不 lease，不 dispatch。
- cmd env/runtime diagnostics test 覆盖 external lease rate-limit attributes。
- `go test ./domain/service ./trigger/job ./app/service ./cmd/agent-runtime -run "TestOutboxAccountRateLimiter|TestOutboxDeliveryWorkerRateLimit|TestWorkQueueExternalLeaseServiceRateLimitsOutbox|TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers|TestOutboxDeliveryWorkerConfigFromEnv"` 通过。
- `go test ./...` 通过。
