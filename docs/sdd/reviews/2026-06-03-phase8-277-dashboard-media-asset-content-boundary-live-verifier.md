# Review: dashboard media asset content boundary live verifier

Spec:
- `docs/sdd/specs/agent-gateway/220-dashboard-media-asset-content-boundary-live-verifier.md`

Implementation summary:
- dashboard runtime-overview fallback 现在会读取并规范化
  `/v1/media-assets/content-diagnostics`
- `media_asset_content` summary/card/detail 与 sampled asset recovery/preflight
  endpoint 会保留到 dashboard payload
- 新增 repo-owned live verifier，直接比对 Go runtime 与 dashboard overview 的
  `media_asset_content` parity，以及 dashboard preflight proxy 与 Go preflight parity
- unified goal verifier 现在把这条证据固定到 top-level artifact、residual
  classification 和 `checks.dashboard_read_models.*`

Tests run:
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_dashboard_media_asset_content_boundary.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-dashboard-media-asset-content-boundary.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

Findings:
- 问题根因不是 Go runtime 缺字段，而是 dashboard fallback 没补
  `media_asset_content`，并在 normalize 阶段丢了 recovery/preflight endpoint
- 当前修复后，dashboard fallback 可在只读前提下保留 content diagnostics 和
  preflight link，且不会执行 recovery side effect

Decision:
- 接受。本切片属于 Go-owned control-plane/read-model parity 收尾，不触碰
  Python AI surfaces

Follow-ups:
- 若后续新增新的 media content control-plane card，沿用同样的 repo-owned
  live parity verifier，而不是只依赖 panel 静态字符串探针
