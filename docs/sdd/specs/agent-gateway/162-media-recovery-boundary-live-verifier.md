# Spec 162: media recovery boundary live verifier

## Status

Accepted for the current iteration.

## Context

`OI-005` 现在已经不是“Go 没有 media recovery control plane”：

- Go 已有 `content-diagnostics`、`content-access-plan`、`content-recovery-plan`、
  approval-bound `content-recovery/preflight`、HTTP/HTTPS recovery executor，
  以及 runtime overview recovery card/detail。
- Python/dashboard 也已有 recovery plan/preflight proxy。

但当前仓库还缺一个 repo-owned live verifier，把“本轮 runtime 到底落在哪个
recovery boundary”稳定读出来。没有这个入口时，`OI-005` 的结论只能靠人工拼接：

1. 当前 registry 里的资产主要是什么 URL scheme；
2. 当前 sample asset 的 access/recovery/preflight 实际 reason 是什么；
3. 当前 boundary 是 HTTP/HTTPS cache executor 候选，还是
   `operator_runtime_config`；
4. dashboard proxy 是否正常；
5. dashboard runtime overview 是否已经把 `media_asset_content_recovery`
   detail 真正透出。

## Decision

新增 repo-owned live verifier：

```text
scripts/verify-media-recovery-boundary.ps1
```

它必须在当前 turn 真实读取：

- `/v1/runtime-overview`
- `/v1/media-assets/content-diagnostics`
- `/v1/media-assets`
- sample asset 的：
  - `/v1/media-assets/content-access-plan`
  - `/v1/media-assets/content-recovery-plan`
  - `/v1/media-assets/content-recovery/preflight`
- dashboard:
  - `/api/dashboard/runtime-overview`
  - `/api/dashboard/media-assets/content-recovery-plan`

并输出单个 JSON，至少包含：

- `runtime_overview`
- `diagnostics`
- `plan_samples`
- `sample_asset`
- `dashboard`
- `checks`
- `conclusion`

`conclusion.category` 允许至少三类：

1. `go_control_plane_live_http_executor_candidates_present`
2. `go_control_plane_live_operator_runtime_config_boundary`
3. `go_control_plane_live_mixed_boundary`

当前 verifier 必须能在 live 数据上明确区分：

- 当前样本主要是 `file://` 平台上传镜像，且因为 content roots 不匹配而
  `forbidden`，所以活跃 boundary 是 `operator_runtime_config`；
- 当前是否存在 HTTP/HTTPS asset，决定 HTTP/HTTPS cache executor 是否在
  当前 registry 里有 live candidate；
- dashboard recovery plan proxy 已可用，但 dashboard runtime overview 若仍没把
  `media_asset_content_recovery` card/detail 透出来，必须显式暴露该 read-model
  gap，而不是把它混在 generic dashboard follow-up 里。

## Out of Scope

- 实现 QQ/Telegram 私有源 session/cookie/token fetch executor；
- 自动后台重试；
- 恢复后自动触发 OCR/VLM/RAG/AI enrichment；
- 修改 Go media recovery executor 的 source-scheme 支持范围；
- 替代 `POST /v1/media-assets/content-recovery` 的真实下载执行。

## Acceptance

- `scripts/verify-media-recovery-boundary.ps1` 可直接返回稳定 JSON。
- unified goal verifier 会复用该脚本结果，而不是人工口头描述 `OI-005`。
- 当前 live 运行态下，verifier 能明确给出：
  - 当前样本是否存在 HTTP/HTTPS asset；
  - 当前 dominant boundary 是否为 `operator_runtime_config`；
  - dashboard recovery plan proxy 是否正常；
  - dashboard runtime overview 是否缺失 `media_asset_content_recovery` detail。
- `DONE.md`、`LIVE_CHECKS.md`、`OPEN_ISSUES.md`、`REMAINING_GO_MIGRATION.md`、
  spec index、ATDD、TDD、review 与本轮 live 结果同步更新。
