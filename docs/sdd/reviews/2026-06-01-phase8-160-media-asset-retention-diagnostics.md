# Phase 8.160 Review: Media Asset Retention Diagnostics

日期：2026-06-01

## 范围

- 新增 Go-owned media asset retention diagnostics。
- 暴露 `GET /v1/media-assets/retention-diagnostics`。
- 支持 asset/filter/timestamp/default_ttl_hours/ephemeral_ttl_hours 参数。
- 补 service 和 HTTP 测试。
- 更新 SDD TODO / DONE / LIVE_CHECKS。

## 架构检查

- Go 负责 media asset registry 的确定性保留策略可见性。
- Python 继续负责媒体下载、OCR/VLM、文件解析、图片理解和语义抽取。
- 本轮 endpoint 只读，`side_effect=none`，不删除 registry、不删除文件、不调用 AI pipeline。
- DDD/六边形分层保持：query 定义视图和 filter，app service 编排诊断，trigger/http 暴露入口。

## 行为

- `permanent` / `keep` / `never`：不进入 cleanup due。
- `ephemeral` / `temp` / `temporary` / `short`：默认 TTL 24 小时，可 query 覆盖。
- 空值、`default`、`default-observed-group`：默认 TTL 30 天，可 query 覆盖。
- 其它 retention 归类为 `unknown`，使用 default TTL 做 advisory。

## 验证

- `go test ./app/service -run MediaAsset`
- `go test ./trigger/http -run MediaAsset`
- `go test ./...`
- `git diff --check`

## 风险

- 这是 advisory diagnostics，不代表清理已执行；真实删除需要后续单独设计 operator approval、mutation audit、dry-run/commit 分离和回滚策略。
- 默认 TTL 是 Go runtime 的观察建议，不应被 Python AI pipeline 当作语义数据保留策略。
