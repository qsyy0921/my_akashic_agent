# Receiver Status Cleanup Stale Live Smoke TDD

## Go

1. `receiverstatusstore.Store`
   - 支持 `DeleteReceiverStatus`
   - 删除后 reopen 仍只剩 active record

2. `ReceiverStatusService`
   - `CleanupStaleReceiverStatuses` 只删 stale receiver
   - active receiver 保留
   - file-backed status store 删除后 reopen 结果一致

3. HTTP handler
   - `POST /v1/receiver-statuses/cleanup-stale` 返回 200
   - body 支持可选 `timestamp` 和 `stale_after_seconds`
   - response 包含 `deleted/remaining/totals`

## Python verifier

1. temp runtime smoke 通过
2. live-runtime inspect 能识别 `heartbeat_stale` receivers
3. unified goal verifier wiring 暴露 receiver-status cleanup current-turn evidence
