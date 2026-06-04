# Phase 8.222 Review: MQ adapter boundary dashboard read model

## What changed

- `plugins/runtime_overview/dashboard.py` now preserves these `queue_backend`
  fields from Go-owned runtime state:
  - `supported_providers`
  - `provider_capabilities`
  - `selected_provider_capability`
- `tests/test_runtime_overview_dashboard_plugin.py` now asserts that the
  normalized dashboard payload retains the capability matrix instead of dropping
  it.
- Added repo-owned live verifier:
  - `scripts/verify-mq-adapter-boundary.ps1`
- Unified goal verifier now includes current-turn `mq_adapter_boundary`
  classification via:
  - `scripts/verify-go-migration-goal.ps1`

## Live evidence

本轮实际运行：

- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-mq-adapter-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

关键 live 结果：

- dashboard `queue_backend` 当前已真实包含：
  - `supported_providers`
  - `provider_capabilities`
  - `selected_provider_capability`
- `verify-mq-adapter-boundary.ps1` 当前返回：
  - `checks.selected_provider_visible=true`
  - `checks.nats_is_recommended_first_backend=true`
  - `checks.redis_and_rabbit_are_planned_only=true`
  - `checks.dashboard_preserves_provider_capabilities=true`
  - `conclusion.status=live_verified`
- unified goal verifier 当前也已把 `mq_adapter_boundary` 归类为：
  - `mq_adapter_boundary_live_verified_with_dashboard_read_model`

## Conclusion

这轮把 MQ adapter boundary 的 dashboard read-model gap 收掉了。

当前更准确的结论是：

- NATS JetStream 仍是唯一已实现、且被推荐的第一外部 MQ；
- Redis Streams / RabbitMQ 仍明确是 planned-only adapter boundary；
- 剩余缺口不再是 visibility/read-model，而是未来若出现明确部署需求时的
  真实 adapter 实现与 cutover。
