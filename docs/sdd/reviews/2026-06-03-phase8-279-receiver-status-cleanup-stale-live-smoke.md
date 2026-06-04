# 2026-06-03 Phase8-279 Receiver Status Cleanup Stale Live Smoke

## 结论

已完成。

## 本轮交付

- 新增 Go mutation：`POST /v1/receiver-statuses/cleanup-stale`
- 新增 file-backed receiver status delete 语义
- 新增 repo-owned temp-runtime live smoke
- 新增 unified goal verifier current-turn evidence：
  - `current_state.receiver_statuses_stopped`
  - `current_state.receiver_statuses_heartbeat_stale`
  - `runtime_invariants.receiver_status_cleanup`
  - `residual_classification.receiver_status_cleanup`

## 已验证

- `go test ./app/service ./trigger/http ./infrastructure/receiverstatusstore`
- `uv run pytest tests/test_verify_receiver_status_cleanup_live_smoke.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-receiver-status-cleanup-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode none`

## 当前 live 结果

- temp runtime smoke: `receiver_status_cleanup_live_verified`
- current runtime inspect:
  - `receiver_statuses_stopped=2`
  - `receiver_statuses_heartbeat_stale=2`
  - 默认未对 live runtime 执行 cleanup

## 风险边界

- cleanup 只删除 stale receiver-status record
- 不启动/停止/reconnect receiver
- 不修改 receiver lease
- 不修改 QQ / Telegram credential
- 不触发 Python AI
