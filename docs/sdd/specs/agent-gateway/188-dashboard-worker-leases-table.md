# 188 Dashboard Worker Leases Table

## Context

`/v1/runtime-overview` 已提供 `worker_leases` card 和 diagnostics，但 dashboard
panel 之前只能靠 raw JSON 看 lease 诊断，不能直接看 checkpoint 前缀、stale
lease 计数和 recent jobs。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` 为 `worker_leases` 增加结构化只读
drilldown，展示：

- `jobs/checkpoints/leaseable jobs/stale leases/workers/stale after`
- `job_type`
- `checkpoint_prefix`
- `status_counts`
- `stale_lease_count`
- `leaseable_count`
- `recent_jobs`
- `latest_job`
- `latest_updated`

同时在 dashboard reader 的 Go overview 归一化路径上增加 fallback：如果旧 payload
缺少 `worker_leases` card，则基于 `summary.worker_leases`、`summary.stale_jobs`、
top-level `worker_leases` 和 `diagnostics` 追加一张只读 card。

## Non-Goals

- 不修改 lease 规则
- 不执行 cleanup 或 takeover
- 不新增 worker control mutation

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
