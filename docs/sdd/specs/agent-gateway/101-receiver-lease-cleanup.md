# Receiver Lease Cleanup

日期：2026-06-01

## 背景

Go `agent-runtime` 已经负责 receiver status 和 receiver lease 的确定性控制面。当前 lease 支持 acquire / renew / release / list，但过期 lease 只会在再次 acquire 或人工 release 时被覆盖或删除。对于 QQ / Telegram 接收端的长时间运行场景，过期 lease 长期残留会让 operator 无法区分“曾经的租约”与“当前可接管状态”。

## 目标

- 在 Go runtime 内新增显式过期 lease cleanup 用例。
- 通过 HTTP endpoint 暴露给 dashboard、运维脚本或 Python supervisor 调用。
- 清理行为只影响 Go runtime receiver lease 状态，不启动/停止接收端，不修改配置，不触发 AI。
- 保持现有 observe-only QQ 群采集、Telegram polling 和 Python receiver 逻辑不变。

## 非目标

- 不增加后台定时 scheduler。
- 不抢占活跃 lease。
- 不接管 Python receiver 进程生命周期。
- 不修改 QQ / Telegram 登录、采集或回复策略。

## 设计

### API

新增：

```text
POST /v1/receiver-leases/cleanup-expired
```

请求体可为空，也可传：

```json
{
  "timestamp": "2026-06-01T00:00:00Z"
}
```

返回：

```json
{
  "deleted": [...],
  "remaining": [...],
  "totals": {
    "deleted": 1,
    "remaining": 1,
    "remaining_active": 1,
    "remaining_expired": 0
  },
  "notes": ["side_effect=runtime_state_only", "expired_receiver_leases_removed"],
  "side_effect": "runtime_state_only"
}
```

### 分层

- `api/dto`：`CleanupExpiredReceiverLeasesRequest`
- `app/command`：`CleanupExpiredReceiverLeasesCommand`
- `app/query`：`ReceiverLeaseCleanupView`
- `app/port/in`：扩展 `ReceiverStatusManager`
- `app/service`：在同一把 mutex 下扫描 expired leases，删除内存态和 repository 记录
- `trigger/http`：新增 handler 并注册 endpoint

### 边界

Go 负责 lease 生命周期状态和持久化清理。Python 仍负责实际 QQ / Telegram receiver 进程、模型调用、图片/文件理解、RAG/Memory 等 AI pipeline。

## 验证

- service test：创建活跃和过期 lease，cleanup 只删除过期 lease。
- repository test：文件态 lease cleanup 后 reopen 不再读到过期 lease。
- HTTP test：`POST /v1/receiver-leases/cleanup-expired` 返回 deleted / remaining summary。
- 回归：`go test ./app/service ./trigger/http`、`go test ./...`、`git diff --check`。
