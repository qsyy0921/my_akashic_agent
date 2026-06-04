# 197 Dashboard RAG Eval Failures Table

## Context

`/v1/runtime-overview` 已提供 `rag_eval_failures` card/detail，但 dashboard
panel 之前仍只能通过 raw JSON 查看这条 read-model。

当前 detail 形状存在两类来源：

- dashboard fixture / fallback 形状：`detail.items`
- 当前 live runtime 形状：`detail.agent_job_metrics`

其中 live runtime 的 `rag_eval_failures` card 主要复用 `agent_job_metrics`
中的 `dead_letters` / `throughput` 信息，因此 panel 需要兼容两种 detail。

## Decision

在 `plugins/runtime_overview/dashboard_panel.ts` / `.js` 为
`rag_eval_failures` 增加结构化只读 drilldown，并兼容两种 detail 形状。

展示内容：

- `sampled jobs`
- `rag eval dead letters`
- `total dead letters`
- `failed events`
- `terminal events`
- recent rag-eval failure 表格：
  - `Job`
  - `Lifecycle`
  - `Quality`
  - `Passed`
  - `Attempt`
  - `Lease Owner`
  - `Error`
  - `Updated At`
- notes

当 `detail.items` 缺失时，panel 应回退到
`detail.agent_job_metrics.dead_letters.recent` 中的 `rag_eval` 项。

同时将这条 read-model 纳入 unified goal verifier 的当前 turn 取证。

## Non-Goals

- 不创建 / 租约 / 重试 `rag_eval` AgentJob
- 不修改 AgentJob metrics、dead-letter、lease 或 worker 状态
- 不触发 Python `rag_eval` 执行器
- 不在 dashboard 增加任何控制逻辑

## Verification

- `GET /v1/runtime-overview`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
