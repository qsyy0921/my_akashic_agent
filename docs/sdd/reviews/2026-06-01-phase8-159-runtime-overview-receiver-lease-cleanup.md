# Phase 8.159 Review: Runtime Overview Receiver Lease Cleanup

日期：2026-06-01

## 范围

- Go runtime overview summary 新增 receiver lease cleanup 可见性字段。
- Receiver Leases card value 改为 active/expired。
- Python dashboard fallback 同步 receiver lease cleanup summary 默认值和 card detail。
- 更新 SDD TODO / DONE / LIVE_CHECKS。

## 架构检查

- Go 继续拥有 receiver lease 生命周期和 cleanup endpoint contract。
- Python dashboard 只做展示适配和 fallback normalization，不执行 cleanup，不管理 receiver lease。
- 本轮只增加只读可见性，没有新增后台 scheduler、control mutation 或自动清理副作用。

## 行为

- `receiver_lease_cleanup_required = receiver_leases_expired > 0`。
- `receiver_lease_cleanup_endpoint = /v1/receiver-leases/cleanup-expired`。
- `Receiver Leases` card value 使用 `active/expired`，expired 大于 0 时状态为 warn。

## 验证

- `go test ./app/service`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `go test ./...`
- `git diff --check`

## 风险

- 该字段只是操作提示，不代表 cleanup 已执行。现场验证必须显式调用 cleanup endpoint 并确认 deleted / remaining 变化。
- Go aggregate 和 Python fallback 都提供 endpoint 字符串；如果未来 endpoint 变更，需要同步更新两侧契约。
