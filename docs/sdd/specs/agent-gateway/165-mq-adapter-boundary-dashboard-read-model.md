# Spec 165: MQ adapter boundary dashboard read model

## Status

Accepted for the current iteration.

## Context

`/v1/queue-backend` 已经具备完整的 MQ provider capability matrix，可以明确区分：

1. 当前 selected provider；
2. `nats_jetstream` 是当前推荐第一外部 MQ；
3. `redis_streams` 与 `rabbitmq` 仍是 planned-only adapter boundary。

但本轮 live 结果暴露出一个 dashboard read-model gap：

1. runtime `/v1/queue-backend` 已返回 `supported_providers`；
2. runtime `/v1/queue-backend` 已返回 `provider_capabilities`；
3. runtime `/v1/queue-backend` 已返回 `selected_provider_capability`；
4. dashboard `/api/dashboard/runtime-overview` 的 `queue_backend` 却没有稳定透传这些字段。

这会让 MQ adapter 的剩余边界在 Python dashboard 层丢失，导致运行态虽然能说明
“NATS JetStream 已实现、Redis/Rabbit 仍 planned-only”，但 dashboard 无法直接把这
一层 read model 交付给 operator。

## Decision

本轮补齐 dashboard `queue_backend` 的只读透传：

1. `plugins/runtime_overview/dashboard.py` 的 `queue_backend` normalize 现在必须保留：
   - `supported_providers`
   - `provider_capabilities`
   - `selected_provider_capability`
2. `provider_capabilities` 内的每个 provider item 必须保持结构化字段，而不是退化成
   原始字符串。
3. unified verifier 增加 repo-owned MQ boundary 检查，直接验证：
   - 当前 selected provider 可见；
   - `nats_jetstream` 仍是 recommended first external MQ；
   - `redis_streams` / `rabbitmq` 仍是 planned-only；
   - dashboard runtime-overview 也能稳定保留 capability matrix。

## Out of Scope

- 实现 Redis Streams adapter；
- 实现 RabbitMQ adapter；
- 变更当前 selected provider；
- 启用 external lease / result-ack cutover；
- 修改 QQ / Telegram 发送策略；
- 恢复 QQ 群发。

## Acceptance

- `GET /v1/queue-backend` 返回：
  - `recommended_first_backend=nats_jetstream`
  - `provider_capabilities` 含 `local/nats_jetstream/redis_streams/rabbitmq`
  - `selected_provider_capability.provider=local`
- `GET /api/dashboard/runtime-overview` 的 `queue_backend` 返回：
  - `supported_providers`
  - `provider_capabilities`
  - `selected_provider_capability`
- `.\scripts\verify-mq-adapter-boundary.ps1` 返回：
  - `checks.selected_provider_visible=true`
  - `checks.nats_is_recommended_first_backend=true`
  - `checks.redis_and_rabbit_are_planned_only=true`
  - `checks.dashboard_preserves_provider_capabilities=true`
  - `conclusion.status=live_verified`
- `.\scripts\verify-go-migration-goal.ps1` 能把 `mq_adapter_boundary` 作为当前 turn 的
  live classification 输出，而不是留给人工解释。
