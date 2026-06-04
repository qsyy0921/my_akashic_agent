# ATDD: Agent Worker Status Stale Cleanup Live Smoke

## Scope

- 验证 Go `agent-worker-statuses` 可以显式清理 stale heartbeat 残留记录，
  且不会误删活跃 worker。

## Preconditions

- 可启动隔离 temp `agent-runtime`
- 当前 repo 的 `127.0.0.1:8780` runtime 可访问
- 不启动新的 Python AI worker，不发送平台消息

## Scenarios

### Scenario 1

- Action:
  在隔离 temp runtime 中上报一条超过 stale threshold 的 synthetic worker
  和一条活跃 worker，然后调用
  `POST /v1/agent-worker-statuses/cleanup-stale`
- Expect:
  stale worker 出现在 `deleted`，活跃 worker 保留在 `remaining`

### Scenario 2

- Action:
  读取 temp runtime 的 `GET /v1/agent-worker-statuses` 和
  `agent-worker-statuses.json`
- Expect:
  stale worker 已从 API 和 state file 中移除，活跃 worker 仍保留

### Scenario 3

- Action:
  在当前 live runtime 上检查 stale worker；若存在，则执行一次受控 cleanup
- Expect:
  cleanup 后 `GET /v1/agent-worker-statuses` 的 `totals.stale=0`，并且
  `/v1/runtime-overview.summary.agent_workers_stale=0`

## Failure Signals

- cleanup 删除了活跃 worker
- cleanup 没有删除 stale worker
- API 视图和 `agent-worker-statuses.json` 不一致
- live runtime cleanup 后 `Agent Workers` 仍保留历史 stale 残留

## Evidence

- `scripts/verify-agent-worker-status-cleanup-live-smoke.ps1` 输出 JSON
- live `POST /v1/agent-worker-statuses/cleanup-stale` 返回
- cleanup 前后的 `/v1/agent-worker-statuses` 与 `/v1/runtime-overview`
