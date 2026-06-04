# TDD: Dashboard QQ Cutover Route Matrix Table

## Scope

- 保护 `qq_cutover_route_matrix` 的 dashboard reader 派生逻辑、panel 资产和
  unified verifier wiring。

## Target Code Paths

- `plugins/runtime_overview/dashboard.py`
- `plugins/runtime_overview/dashboard_panel.ts`
- `plugins/runtime_overview/dashboard_panel.js`
- `scripts/verify-go-migration-goal.ps1`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| queue backend route gates preserved | unit | `outbox_execution_owner/scope` 与 `outbox_allowed_kinds*` 不会在 normalization 中丢失 |
| derived route matrix totals | integration | summary/card/top-level payload 正确给出 5/5/3/5 与 `qq_group_send_enabled=false` |
| panel static assets | unit | JS 资产包含 `QQ Cutover Route Matrix` 和四类 route 表字面串 |
| unified verifier wiring | integration | `dashboard_read_models.qq_cutover_route_matrix_table=true` |

## Required Automated Tests

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`

## Deferred Coverage

- 不在这条 TDD 里重跑 native rich-media probe。
- 不在这条 TDD 里验证真实 QQ 发送，只验证 read-model 与 verifier 证据。
