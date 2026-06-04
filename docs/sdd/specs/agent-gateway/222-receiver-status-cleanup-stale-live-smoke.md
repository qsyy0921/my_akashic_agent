# 222. Receiver Status Cleanup Stale Live Smoke

## 背景

Go runtime 已能把过期 receiver heartbeat 投影成 `status=stopped` /
`reason=heartbeat_stale`，但此前没有显式 mutation 删除这些 stale
receiver-status 残留，也缺少 repo-owned smoke 去证明“只删 stale，不误删 active”。

## 目标

1. 新增 `POST /v1/receiver-statuses/cleanup-stale`
2. 仅删除满足 stale heartbeat 条件的 receiver-status record
3. 返回 `deleted/remaining/totals/notes/side_effect=runtime_state_only`
4. 提供 repo-owned temp-runtime live smoke，并把 current-turn evidence 接进
   unified goal verifier

## 非目标

- 不启动、停止、重连任何 QQ / Telegram receiver
- 不修改 `receiver-leases`
- 不修改 runtime-config 或 credential
- 不把 cleanup 自动接进 runtime-overview 或 dashboard fallback
- 不迁移任何 Python AI surface

## 设计

### 1. Cleanup 命令与输出

新增：

- `command.CleanupStaleReceiverStatusesCommand`
- `query.ReceiverStatusCleanupView`
- `dto.CleanupStaleReceiverStatusesRequest`

请求字段：

- `receiver_id`：可选，只清理单条 receiver
- `timestamp`：可选，默认当前 UTC
- `stale_after_seconds`：可选，覆盖 service 默认 stale threshold

返回字段：

- `deleted`
- `remaining`
- `totals.deleted`
- `totals.remaining`
- `totals.remaining_stopped`
- `notes=["side_effect=runtime_state_only","stale_receiver_statuses_removed"]`

### 2. Stale 判定必须复用现有 heartbeat 语义

cleanup 与 list path 共享同一套 stale 条件：

- 只处理 `starting` / `connected`
- `updated_at` 超过 stale threshold
- 投影后的 reason 为 `heartbeat_stale`

### 3. Repository 删除能力

`ReceiverStatusRepository` 新增 `DeleteReceiverStatus(ctx, receiverID)`。

file-backed `receiverstatusstore.Store` 需要支持删除后 flush 回
`receiver-statuses.json`。

### 4. Live smoke

新增：

- `scripts/verify_receiver_status_cleanup_live_smoke.py`
- `scripts/verify-receiver-status-cleanup-live-smoke.ps1`

temp runtime smoke 至少证明：

1. stale receiver report accepted
2. active receiver report accepted
3. pre-cleanup stale receiver 在 API 中可见且 reason=heartbeat_stale
4. cleanup 返回 `deleted=1`
5. stale receiver 从 API 消失
6. active receiver 仍保留
7. `receiver-statuses.json` 与 API 一致

同时允许 inspect current live runtime，但默认不直接修改 live runtime。

## 验收

1. `go test ./app/service ./trigger/http ./infrastructure/receiverstatusstore` 通过
2. `uv run pytest tests/test_verify_receiver_status_cleanup_live_smoke.py ...` 通过
3. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-receiver-status-cleanup-live-smoke.ps1`
   返回 `conclusion.status=live_verified`
4. `.codex-goal-verifier.json` 当前 turn 暴露：
   - `current_state.receiver_statuses_stopped`
   - `current_state.receiver_statuses_heartbeat_stale`
   - `runtime_invariants.receiver_status_cleanup`
   - `residual_classification.receiver_status_cleanup`
