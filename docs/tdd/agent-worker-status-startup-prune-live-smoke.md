# Agent Worker Status Startup Prune TDD

## Go

1. `services/agent-runtime/app/service/agent_worker_status_service.go`
   - repository load 时对 stale worker 做启动即 prune
2. `services/agent-runtime/app/service/agent_worker_status_service_test.go`
   - 增加 repository load prune regression test

## Scripts / Python

1. `scripts/verify_agent_worker_status_startup_prune_live_smoke.py`
   - 预写 stale + active seed state，启动 temp runtime，校验 startup prune
2. `scripts/verify-agent-worker-status-startup-prune-live-smoke.ps1`
   - 提供 repo-owned smoke 入口
3. `tests/test_verify_agent_worker_status_startup_prune_live_smoke.py`
   - 校验 verifier shape 与核心 checks
4. `tests/test_verify_go_migration_goal_script.py`
   - 校验 unified verifier wiring 与 `current_state.agent_workers_*`
