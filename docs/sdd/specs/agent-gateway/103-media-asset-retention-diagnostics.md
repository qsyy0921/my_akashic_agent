# Media Asset Retention Diagnostics

日期：2026-06-01

## 背景

Go runtime 已经持有媒体资产 registry、内容安全访问和 content diagnostics。`MediaAsset.Retention` 目前只是登记字段，没有稳定的只读视图说明哪些资产按保留策略已经进入清理候选。媒体文件和图片是 QQ 群观察系统的重要基础设施，清理逻辑必须先可见、可审计，再考虑删除。

## 目标

- 新增 Go-owned media asset retention diagnostics。
- 按 retention policy、asset age 和 cutoff 输出只读 advisory。
- 支持按现有 media asset filter 查询：channel、account、conversation、source message、kind、limit。
- 暴露 HTTP endpoint 供 dashboard/operator 观察。
- 不删除 registry，不删除本地文件，不调用 OCR/VLM/文件解析。

## 非目标

- 不实现真实删除。
- 不移动或压缩媒体文件。
- 不改变 `/v1/media-assets/{id}/content` 访问策略。
- 不把 Python 的图片理解、OCR、文件语义抽取迁到 Go。

## API

```text
GET /v1/media-assets/retention-diagnostics
```

可选参数：

- `asset_id`
- `limit`
- `channel_kind` / `kind`
- `account_id`
- `conversation_id`
- `conversation_type`
- `source_message_id`
- `source_message_id_suffix`
- `asset_kind`
- `timestamp`
- `default_ttl_hours`
- `ephemeral_ttl_hours`

返回只读结构：

```json
{
  "items": [
    {
      "asset_id": "asset:...",
      "retention": "default-observed-group",
      "retention_class": "default",
      "cleanup_due": false,
      "age_seconds": 3600,
      "ttl_seconds": 2592000,
      "cleanup_after": "2026-06-30T00:00:00Z",
      "cleanup_reason": "media_asset_retention_not_due"
    }
  ],
  "totals": {
    "assets": 1,
    "cleanup_due": 0,
    "permanent": 0,
    "default": 1,
    "ephemeral": 0,
    "unknown": 0
  },
  "side_effect": "none"
}
```

## Policy

本轮只做 advisory policy：

- `permanent` / `keep` / `never`：不进入 cleanup due。
- `ephemeral` / `temp` / `temporary` / `short`：默认 24 小时。
- 其它空值、`default`、`default-observed-group`：默认 30 天。

TTL 可通过 query 覆盖，用于测试、dry-run 或不同环境观察。

## Go / Python 边界

- Go：media asset registry、retention diagnostics、HTTP API、前端稳定数据接口。
- Python：下载/补充媒体文件、OCR/VLM/文件解析、AI 语义抽取、图片生成和实验型策略。

## 验证

- service test：覆盖 permanent/default/ephemeral 的 due 分类。
- HTTP test：覆盖 endpoint、timestamp 和 TTL 参数。
- 回归：`go test ./app/service ./trigger/http`、`go test ./...`、`git diff --check`。
