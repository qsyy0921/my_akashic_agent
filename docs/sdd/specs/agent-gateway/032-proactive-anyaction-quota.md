# 032 Proactive AnyAction Quota

日期：2026-05-31

## 目标

把主动推送 AnyAction 的确定性 quota 状态迁移到 Go `agent-runtime`：每日窗口、窗口 rollover、已用次数和最近 action 时间由 Go 持久化和审计。Python 仍保留 AnyAction 的概率计算、idle factor、time factor 和主动推送策略判断。

## 边界

- Go `domain`：`ProactiveAnyActionQuota` 表示当前 quota window。
- Go `app`：`SnapshotAnyActionQuota` 负责按时区和 reset hour 做 rollover；`RecordAnyAction` 负责记录一次 action。
- Go `infrastructure`：复用 `proactivestate` JSON 文件态，新增 `anyaction_quotas` 字段，兼容旧文件。
- Go `trigger`：
  - `GET /v1/proactive/anyaction/quota`
  - `POST /v1/proactive/anyaction/actions`
- Python：`AgentRuntimeAnyActionQuotaStore` 优先调用 Go；失败时回退原 `QuotaStore` JSON 文件。

## 不做

- 不把主动推送的概率、内容判断、LLM 决策迁移到 Go。
- 不改变 `AnyActionGate.should_act` 的计算公式。
- 不触发发送，不创建 outbox，不影响 QQ/Telegram 接收链路。

## API

```http
GET /v1/proactive/anyaction/quota?quota_key=default&reset_hour=12&timezone=Asia%2FShanghai&timestamp=2026-05-30T03:00:00Z
```

响应：

```json
{
  "quota_key": "default",
  "window_key": "2026-05-29@12@Asia/Shanghai",
  "next_reset_at": "2026-05-30T04:00:00Z",
  "used": 0,
  "last_action_at": "",
  "found": false,
  "side_effect": "runtime_state_rollover"
}
```

```http
POST /v1/proactive/anyaction/actions
```

```json
{
  "quota_key": "default",
  "reset_hour": 12,
  "timezone": "Asia/Shanghai",
  "timestamp": "2026-05-30T03:01:00Z"
}
```

## 验收

- Go service 测试覆盖 rollover、record 和 last action 保留。
- Go store 测试覆盖 `anyaction_quotas` 文件持久化。
- HTTP handler 测试覆盖 snapshot/action API。
- Python runtime proactive state 测试覆盖 Go quota snapshot/action 和 fallback JSON 兼容写入。
