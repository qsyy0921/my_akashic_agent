# Receiver Status Cleanup Stale Live Smoke ATDD

## 场景

当 Go runtime 中同时存在一条 stale receiver-status 和一条 active
receiver-status 时，`POST /v1/receiver-statuses/cleanup-stale` 应只删除 stale
记录。

## 验收步骤

1. 在 temp `agent-runtime` 中上报：
   - 一条 5 分钟前的 `connected` QQ receiver
   - 一条当前时间的 `connected` Telegram receiver
2. 调用 `GET /v1/receiver-statuses`
3. 断言 stale QQ receiver 已被投影成 `stopped + heartbeat_stale`
4. 调用 `POST /v1/receiver-statuses/cleanup-stale`
5. 再次调用 `GET /v1/receiver-statuses`
6. 断言：
   - stale receiver 进入 `deleted`
   - active receiver 仍在 `remaining`
   - `receiver-statuses.json` 与 API 一致
7. inspect 当前 live runtime，记录是否仍存在 `heartbeat_stale`
   receiver-status，但默认不对 live runtime 做 cleanup
