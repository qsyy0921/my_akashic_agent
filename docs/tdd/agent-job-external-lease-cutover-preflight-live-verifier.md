# TDD: Agent Job External Lease Cutover Preflight Live Verifier

## Test Scope

- `tests/test_verify_agent_job_external_lease_cutover_preflight.py`

## Assertions

1. `_build_result(...)` 在两个场景 checks 全部通过时返回：
   - `status=live_verified`
   - `category=repo_owned_temp_runtime_cutover_preflight_live_verified`
2. 任一 preflight check 失败时返回：
   - `status=verification_failed`
   - `category=agent_job_external_lease_cutover_preflight_failed`
3. runtime / docker 启动异常时返回：
   - `status=error`
   - `category=agent_job_external_lease_cutover_preflight_error`

## Live Smoke

- `uv run python scripts/verify_agent_job_external_lease_cutover_preflight.py --repo-root E:\agent\my-akashic_agent`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-preflight.ps1`
