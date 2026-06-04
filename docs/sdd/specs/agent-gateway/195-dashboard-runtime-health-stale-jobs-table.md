# 195 Dashboard Runtime Health Stale Jobs Table

## Context

`/v1/runtime-overview` 已提供 `runtime_health` 和 `stale_jobs` card/detail，但
dashboard panel 之前仍只能通过 raw JSON 阅读 runtime health snapshot 和 stale
job diagnostics。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` / `.js` 为
`runtime_health`、`stale_jobs` 增加结构化只读 drilldown，展示：

- `runtime_health`
  - `status/errors/health fields`
  - health snapshot field/value 表
  - health error 表
- `stale_jobs`
  - `jobs/checkpoints/leaseable jobs/stale leases/workers/stale after`
  - worker diagnostics 的
    `job type/checkpoint prefix/status counts/stale leases/leaseable/latest job/latest updated`
  - stale job sample 的
    `job/job type/status/lease owner/attempts/latest updated`

同时把这两条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不执行 runtime recovery
- 不执行 stale lease cleanup / takeover / retry
- 不修改 worker、queue owner、checkpoint 或 job 状态
- 不触发 Python AI

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
