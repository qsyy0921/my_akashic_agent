# ATDD: MQ adapter boundary dashboard read model

## Scope

- dashboard `/api/dashboard/runtime-overview` 对 Go-owned `queue_backend`
  capability matrix 的只读透传；
- repo-owned verifier 对当前 MQ adapter boundary 的 live 归类。

## Preconditions

- Go runtime 在 `http://127.0.0.1:8780` 可访问。
- Python dashboard 在 `http://127.0.0.1:2236` 可访问。
- repo 内脚本可执行：
  - `scripts/verify-mq-adapter-boundary.ps1`
  - `scripts/verify-go-migration-goal.ps1`

## Scenarios

### Scenario 1

- Action:
  请求 `GET /v1/queue-backend` 与
  `GET /api/dashboard/runtime-overview`。
- Expect:
  dashboard payload 的 `queue_backend` 同时包含：
  - `supported_providers`
  - `provider_capabilities`
  - `selected_provider_capability`

### Scenario 2

- Action:
  运行 `.\scripts\verify-mq-adapter-boundary.ps1`。
- Expect:
  输出中：
  - `checks.selected_provider_visible=true`
  - `checks.nats_is_recommended_first_backend=true`
  - `checks.redis_and_rabbit_are_planned_only=true`
  - `checks.dashboard_preserves_provider_capabilities=true`
  - `conclusion.status=live_verified`

### Scenario 3

- Action:
  运行 `.\scripts\verify-go-migration-goal.ps1`。
- Expect:
  unified verifier 中：
  - 存在 `mq_adapter_boundary`
  - `residual_classification.mq_adapter_boundary.status=live_verified`
  - 结论明确为 capability/read-model 已补齐，而不是 adapter 已扩展到 Redis/Rabbit

## Failure Signals

- dashboard `queue_backend` 丢失 `provider_capabilities`；
- dashboard `queue_backend` 丢失 `selected_provider_capability`；
- verifier 仍把 MQ boundary 归类成 dashboard read-model gap；
- dashboard 把 Redis/Rabbit 错误标成已实现。

## Evidence

- `GET http://127.0.0.1:8780/v1/queue-backend`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `.\scripts\verify-mq-adapter-boundary.ps1`
- `.\scripts\verify-go-migration-goal.ps1`
