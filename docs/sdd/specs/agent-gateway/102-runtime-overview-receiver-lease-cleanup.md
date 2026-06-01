# Runtime Overview Receiver Lease Cleanup Visibility

日期：2026-06-01

## 背景

Go runtime 已有 receiver lease list 和显式 cleanup endpoint：

```text
GET  /v1/receiver-leases
POST /v1/receiver-leases/cleanup-expired
```

`/v1/runtime-overview` 当前只暴露 `receiver_leases_active` 和 `receiver_leases_expired`。前端或 operator 可以推断是否需要 cleanup，但缺少稳定字段表达“是否建议清理”和“清理入口在哪里”。这会让 dashboard 逻辑重复 Go 侧规则。

## 目标

- 在 Go runtime overview summary 中新增：
  - `receiver_lease_cleanup_required`
  - `receiver_lease_cleanup_endpoint`
- Receiver Leases card value 从单一 active 数改为 `active/expired`，更直接暴露过期租约。
- Python dashboard fallback 同步规范化这些字段，保持 Go aggregate 不可用时的数据契约稳定。

## 非目标

- 不自动调用 cleanup endpoint。
- 不新增后台定时 cleanup worker。
- 不启动/停止 QQ、Telegram receiver。
- 不修改 receiver lease acquire / renew / release 语义。

## Go / Python 边界

- Go：确定性 runtime lease 状态、cleanup 可见性、endpoint contract。
- Python：dashboard 展示适配和 fallback；不复制 cleanup 业务规则之外的执行逻辑，不直接管理 lease 生命周期。

## 验证

- Go runtime overview service test 覆盖 cleanup summary 和 card value/status。
- Python dashboard plugin test 覆盖 fallback summary 默认值和 runtime aggregate passthrough。
- 回归命令：
  - `go test ./app/service`
  - `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q`
  - `git diff --check`
