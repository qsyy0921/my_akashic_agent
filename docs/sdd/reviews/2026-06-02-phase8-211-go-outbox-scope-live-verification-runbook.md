# Phase 8.211 Review - Go Outbox Scope Live Verification Runbook

Date: 2026-06-02

## Summary

新增 `scripts/verify-go-outbox-scope-live.ps1`，把当前 Go local outbox worker
已放开的成功路由和应继续被 gate 的路由固化为 repo-owned live smoke。

同时把该 live smoke 以显式开关并入
`scripts/verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke`，让后续每轮
goal 更新既可以默认只读，也可以在需要时拉起真实 outbox scope 验证。

## What Changed

- 新增：
  - `scripts/verify-go-outbox-scope-live.ps1`
- 更新：
  - `scripts/verify-go-migration-goal.ps1`

新增 live smoke 覆盖：

- 第一账号 `private file` -> 预期成功
- 第二账号 `group file` -> 预期成功
- 第一账号 `group file` -> 预期 gated
- 第一账号 `private image` -> 预期 gated

## Verification

已运行：

```powershell
.\scripts\verify-go-outbox-scope-live.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

本轮 live 结果：

- `first_account_private_file` -> `succeeded`
- `second_account_group_file` -> `succeeded`
- `first_account_group_file_gated` -> `queued/attempts=0`
- `first_account_private_image_gated` -> `queued/attempts=0`
- 统一 goal 脚本中的 `qq_outbox_cutover.conclusion` 已提升为
  `partial_go_default_owner_live_verified`
- 同轮原生 probe 仍证明第一账号 `3219982` group rich-media 最终
  `rich media transfer failed`

## Risks / Follow-up

- 该脚本会真实发送 file smoke，不适合默认在所有只读验证里自动执行。
- rich-media 平台 blocker 仍未解除；当前新增的是“partial cutover 真实边界”的稳定证据，不是 image/group-file 问题的修复。
