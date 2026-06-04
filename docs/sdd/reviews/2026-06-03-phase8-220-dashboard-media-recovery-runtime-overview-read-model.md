# Phase 8.220 Review: dashboard media recovery runtime-overview read model

## What changed

- `plugins/runtime_overview/dashboard.py` now:
  - normalizes top-level `media_asset_content_recovery` detail from Go runtime
    overview;
  - synthesizes `Media Content Recovery` card when Go runtime cards omit it;
  - preserves a muted/unknown fallback field/card instead of dropping the field;
  - uses a longer dedicated timeout for `/v1/runtime-overview` so dashboard does
    not silently fall back on this machine's live runtime.
- `tests/test_runtime_overview_dashboard_plugin.py` now covers both:
  - Go aggregate path with media recovery detail/card;
  - fallback path with muted/unknown media recovery field/card.

## Live evidence

本轮实际运行：

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `GET http://127.0.0.1:8780/v1/runtime-overview`
- `.\scripts\verify-media-recovery-boundary.ps1`
- `.\scripts\verify-go-migration-goal.ps1`

关键 live 结果：

- dashboard runtime-overview 现在返回：
  - `card_present=true`
  - `detail_present=true`
  - `detail_reason=media_asset_content_recovery_audit_ready`
  - `partial=false`
- `verify-media-recovery-boundary.ps1` 现在返回：
  - `dashboard.runtime_overview_has_media_content_recovery_card=true`
  - `dashboard.runtime_overview_has_media_content_recovery_detail=true`
  - `checks.dashboard_runtime_overview_missing_media_recovery_detail=false`
- unified goal verifier 继续只保留：
  - `telegram_token_missing`
  - `qq_image_native_platform_blocker_unresolved`

## Conclusion

这轮把 `OI-005` 里 media recovery 的 dashboard read-model gap 真正收口了。

当前 media recovery 剩余问题已经收敛成：

- 当前 registry 仍没有 HTTP/HTTPS live candidate；
- 当前主边界仍是 `file:// + content roots mismatch` 的
  `operator_runtime_config`；
- 平台私有源 session/token/cookie fetch executor 仍未统一完成。

也就是说，dashboard `/api/dashboard/runtime-overview` 对
`media_asset_content_recovery` 的缺口已经不再是 open blocker。
