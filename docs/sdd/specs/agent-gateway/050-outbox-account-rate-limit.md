# 050 Outbox Account Rate Limit

Date: 2026-05-31

## 背景

上一切片已经把 outbox account pressure 诊断接入 Go runtime。下一步需要让
Go local outbox delivery worker 具备账号级发送节流能力，避免同一个 QQ/Telegram
账号在短时间内连续发送，降低 bot 互聊、平台限速和附件发送堆积时的风险。

本切片只覆盖 Go local outbox delivery worker。它默认关闭，且只有在
`AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` 并显式配置账号节流参数时才生效。
不改 Python AI worker、prompt、模型调用、RAG/Memory 算法，也不切换真实平台发送。

## Go / Python 边界

Go 负责：

- 计算 outbox delivery 的账号 key：`channel_kind:account_id`。
- 在 lease 前跳过处于节流中的账号，避免 delivery 被拿走后卡在 dispatching lease。
- 在 worker 本地记录最近 dispatch attempt，用于 min interval / window limit 判断。
- 在 runtime worker diagnostics 中暴露节流配置。

Python 负责：

- 不参与本切片。
- 继续负责 AI 内容生成、工具选择、图片生成、Memory/RAG 抽取和 provider fallback。

## 范围

- `LeaseNextOutboxCommand` 增加 `BlockedAccountKeys`。
- `OutboxRepository.FindLeaseableOutboxDelivery` 支持 `OutboxLeaseFilter`。
- local worker 增加可选账号节流配置：
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MIN_INTERVAL_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_WINDOW_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MAX_PER_WINDOW`
- 只要账号处于节流窗口，worker 在 lease 前把该账号加入 blocked list。
- 如果队列里还有其它账号的 delivery，worker 应继续租约其它账号，避免队头阻塞。

## 不做

- 不对外部 NATS `external_lease` outbox executor 做生产 cutover。
- 不新增持久化 rate-limit state；本切片是单进程 worker 的 runtime guard。
- 不改 Python 兼容 outbox worker。
- 不把账号节流升级为平台权限或风控策略。

## 验收

- Go outbox service test 覆盖 blocked account key 会跳过对应 delivery。
- Go outbox worker test 覆盖账号节流后不会 dispatch、不会 mark failed、不会占用 blocked delivery。
- Go env/runtime diagnostics test 覆盖节流配置解析和 worker attributes。
- `go test ./app/service ./trigger/job ./cmd/agent-runtime -run "TestOutbox.*Rate|TestOutbox.*Blocked|TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers|TestOutboxDeliveryWorkerConfigFromEnv"` 通过。
- `go test ./...` 通过。
