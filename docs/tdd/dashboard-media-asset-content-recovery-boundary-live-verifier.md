# TDD: Dashboard Media Asset Content Recovery Boundary Live Verifier

## 目标测试

- `tests/test_verify_dashboard_media_asset_content_recovery_boundary.py`
  - dashboard 与 runtime 一致时返回 live parity
  - summary 漂移时能检测失败
  - `_build_result()` 正确区分 `live_verified / verification_failed / error`

- `tests/test_runtime_overview_dashboard_plugin.py`
  - runtime overview 聚合不可用时，dashboard fallback 仍能从 control-mutation audit 合成
    `media_asset_content_recovery_audit_ready`
  - fallback card/detail 不再退回 `unknown/muted`

- `tests/test_verify_go_migration_goal_script.py`
  - unified verifier 已接入 dashboard media recovery boundary verifier
  - `media_asset_content_recovery_table` 已进入 canonical artifact

