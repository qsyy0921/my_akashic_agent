# Phase 8.219 Review: media recovery boundary live verifier

## What changed

- 新增 `scripts/verify-media-recovery-boundary.ps1`
- `scripts/verify-go-migration-goal.ps1` 现在聚合 `media_recovery`
- 新增本轮 `SPEC-162`、ATDD、TDD

## Live evidence

本轮实际运行：

- `.\scripts\verify-media-recovery-boundary.ps1`
- `.\scripts\verify-go-migration-goal.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

关键 live 结果：

- 当前 sampled assets 全为 `file://`
- `content_diagnostics.totals` 为：
  - `assets=120`
  - `forbidden=120`
- sample asset 当前：
  - `access_plan.reason=media_asset_content_forbidden`
  - `recovery_plan.reason=media_asset_content_recovery_fix_content_roots`
  - `preflight.reason=missing_approval_id`
  - `preflight.executor_scope=operator_runtime_config`
- `checks.has_http_https_assets=false`
- `checks.operator_runtime_config_boundary_dominant=true`
- dashboard:
  - `dashboard_recovery_plan_proxy_ready=true`
  - `runtime_overview_has_media_content_recovery_card=false`
  - `runtime_overview_has_media_content_recovery_detail=false`

## Conclusion

这轮把 `OI-005` 从“已有 media recovery 组件但缺当前 turn 边界证据”推进成了 repo-owned live verifier。

当前最准确的边界是：

- Go media recovery control plane 是 live 的；
- 当前 registry 样本主要落在 `file:// + content roots mismatch`，
  所以活跃边界是 `operator_runtime_config`；
- HTTP/HTTPS cache recovery executor 已存在，但当前 registry 没有 live candidate；
- dashboard plan/preflight proxy 是 live 的；
- dashboard runtime overview 对 `media_asset_content_recovery` 的 card/detail
  仍有 read-model gap。
