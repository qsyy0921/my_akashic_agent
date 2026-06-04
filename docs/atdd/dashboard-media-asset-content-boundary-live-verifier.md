# Dashboard Media Asset Content Boundary Live Verifier ATDD

## 场景

当 Go runtime overview 可读，而 dashboard overview 因聚合超时走 fallback 时，
operator 仍需要在 dashboard 中看到 `Media Asset Content` 的 summary/card/detail
和 sampled asset 的 recovery/preflight 链接。

## 验收步骤

1. 调用 Go `/v1/runtime-overview`
2. 调用 dashboard `/api/dashboard/runtime-overview`
3. 选择一个 sampled `media_asset_content` asset
4. 分别调用：
   - Go `/v1/media-assets/content-recovery/preflight`
   - dashboard `/api/dashboard/media-assets/content-recovery/preflight`
5. 比较 summary/card/detail/sample/preflight proxy 是否一致

## 通过条件

- dashboard overview 暴露 `media_asset_content_diagnostics`
- dashboard `Media Asset Content` card 与 Go card 一致
- sampled asset 的 `content_recovery_preflight_endpoint` 未丢失
- dashboard preflight proxy 与 Go preflight 结果一致
- 所有相关 detail / preflight 都保持 `side_effect=none`
