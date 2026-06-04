# ATDD: Agent Worker Status Fencing Live Smoke

## Scope

- 验证 Go `agent-worker-statuses` 的默认 lease fencing 和显式 takeover 边界。

## Preconditions

- 本机可启动隔离 temp `agent-runtime`
- 不复用长期运行的 `http://127.0.0.1:8780`
- 不启动真实 Python worker，不发送平台消息

## Scenarios

### Scenario 1

- Action:
  对同一 synthetic `worker_id` 先上报 `instance_a`
- Expect:
  `POST /v1/agent-worker-statuses/report` 返回 accepted，`GET /v1/agent-worker-statuses`
  中能读到 `instance_a`

### Scenario 2

- Action:
  对同一 `worker_id` 再上报 `instance_b`，不带
  `replace_existing_instance_id`
- Expect:
  返回 `HTTP 409`，并在冲突文本中包含 `existing_instance_id=instance_a`

### Scenario 3

- Action:
  再次上报 `instance_b`，这次显式带
  `replace_existing_instance_id=instance_a`
- Expect:
  takeover 成功，`GET /v1/agent-worker-statuses` 与
  `agent-worker-statuses.json` 都切到 `instance_b`

## Failure Signals

- 第二实例没有返回 409
- 409 文本不含 `existing_instance_id`
- 未经匹配 replace 的 takeover 被错误接受
- takeover 成功后内存视图与 state file 不一致

## Evidence

- `scripts/verify-agent-worker-status-fencing-live-smoke.ps1` 输出 JSON
- temp runtime `agent-worker-statuses.json`
- unified goal verifier 输出中的 `runtime_invariants.agent_worker_status_fencing`
