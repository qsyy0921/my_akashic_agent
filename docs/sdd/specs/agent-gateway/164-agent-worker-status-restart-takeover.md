# 164 Agent Worker Status Restart Takeover

## Context

Go 已经通过 `instance_id + lease_until` 对 Python AI worker status 做了
lease/fencing，能阻止同一 `worker_id` 的活跃实例被另一个进程覆盖。

但在本地 `main.py` 重启时，旧 Python 进程已经退出，而 Go 里的
worker-status lease 还没过期。新的 Python 主进程如果直接上报：

- `akashic-python-worker:image`
- `akashic-python-worker:knowledge`
- `akashic-python-worker:outbox`

会连续收到 `HTTP 409 agent worker status lease conflict`，直到旧 lease 自然过期。

这不是想要的 fencing 语义。这里的正确目标不是放宽冲突检查，而是允许
**新进程在证明旧实例已经消失后，做一次受控 takeover**。

## Requirements

1. Go `/v1/agent-worker-statuses/report` 保持默认冲突拒绝：
   - 活跃 lease + 不同 `instance_id` 仍返回 409。
2. 仅当调用方显式提供 `replace_existing_instance_id` 且它与当前记录中的
   `existing_instance_id` 完全匹配时，Go 才允许覆盖当前记录。
3. Python `AgentWorkerStatusReporter` 只在以下条件下做一次 takeover retry：
   - 收到 409 conflict；
   - 能从错误文本解析 `existing_instance_id`；
   - 能从该 instance id 安全提取旧 PID；
   - 确认旧 PID 已不存在；
   - 重试时显式带上 `replace_existing_instance_id`。
4. 如果旧 PID 仍存活、instance id 无法解析、不是 409、或 takeover retry 再失败，
   仍要保留 warning，不允许静默吞掉真实冲突。
5. restart 后的 live 证据必须证明：
   - fresh log 中没有新的 `agent worker status lease conflict`；
   - `image/knowledge/outbox` 三个 active worker 的 `instance_id` 已切到当前
     Python 主进程 PID；
   - `/v1/agent-worker-statuses` 仍保持可读，QQ 群发关闭与其它既有运行约束不被破坏。

## Non-Goals

- 不改变 AgentJob lease token 语义。
- 不改变 Python worker job execution owner。
- 不把 worker status registry 变成自动调度器。
