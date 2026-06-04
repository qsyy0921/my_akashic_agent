# ATDD: media recovery boundary live verifier

## Goal

把 `OI-005 media recovery` 的当前运行边界变成 repo-owned live 证据，而不是人工口头分类。

## Acceptance Scenarios

1. 运行 `.\scripts\verify-media-recovery-boundary.ps1`
   - 返回单个 JSON。
   - 包含 `runtime_overview`、`diagnostics`、`plan_samples`、`sample_asset`、
     `dashboard`、`checks`、`conclusion`。

2. 当前 runtime 样本以 `file://` 为主且 content roots 不匹配时
   - `checks.has_file_assets=true`
   - `checks.all_sampled_assets_forbidden=true`
   - `checks.operator_runtime_config_boundary_dominant=true`
   - `conclusion.category=go_control_plane_live_operator_runtime_config_boundary`

3. sample asset recovery 证据
   - `sample_asset.recovery_plan.reason=media_asset_content_recovery_fix_content_roots`
   - `sample_asset.preflight.reason=missing_approval_id`
   - `sample_asset.preflight.executor_scope=operator_runtime_config`

4. dashboard 证据分离
   - `dashboard.dashboard_recovery_plan_proxy_ready=true`
   - 若 `/api/dashboard/runtime-overview` 未透出 `media_asset_content_recovery`
     card/detail，则 `checks.dashboard_runtime_overview_missing_media_recovery_detail=true`

5. unified goal verifier 集成
   - `.\scripts\verify-go-migration-goal.ps1` 输出新增 `media_recovery`
   - `residual_classification.media_recovery_private_source_executor`
     与独立 verifier `conclusion` 一致
