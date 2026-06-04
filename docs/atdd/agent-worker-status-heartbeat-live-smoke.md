# ATDD: Agent Worker Status Heartbeat Live Smoke

## Scope

- 验证 Go `agent-worker-statuses` 对同一 worker instance 的长任务 heartbeat
  renewal 边界。

## Preconditions

- 本机可启动隔离 temp `agent-runtime`
- 不复用长期运行的 `http://127.0.0.1:8780`
- 不启动真实 Python worker，不发送平台消息

## Scenarios

### Scenario 1

- Action:
  对同一 synthetic `worker_id` 和同一 `instance_id` 连续上报三次 `running`
  heartbeat
- Expect:
  三次 `POST /v1/agent-worker-statuses/report` 都返回 accepted

### Scenario 2

- Action:
  读取 `GET /v1/agent-worker-statuses`
- Expect:
  对应 worker record 的 `updated_at` 与 `lease_until` 会随 heartbeat 持续推进，
  且最终 `lease_active=true`、`stale=false`

### Scenario 3

- Action:
  读取 temp `agent-worker-statuses.json`
- Expect:
  state file 中该 worker 的最终 `updated_at` / `lease_until` 与 API 视图一致

## Failure Signals

- 任一 heartbeat 未被接受
- `updated_at` 没有推进
- `lease_until` 没有推进
- final record 被错误标记成 stale
- API 视图与 state file 不一致

## Evidence

- `scripts/verify-agent-worker-status-heartbeat-live-smoke.ps1` 输出 JSON
- temp runtime `agent-worker-statuses.json`
- unified goal verifier 输出中的 `runtime_invariants.agent_worker_status_heartbeat`
