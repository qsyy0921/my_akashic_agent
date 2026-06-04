# ATDD: Media Asset Content Recovery HTTP Executor Live Smoke

## Scenario

As an operator, I need a repo-owned smoke that proves the Go HTTP/HTTPS media
recovery executor can complete an approval-bound recovery loop without relying
on QQ/Telegram private-source credentials or Python AI.

## Acceptance

1. Run
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-media-asset-content-recovery-live-smoke.ps1`.
2. Confirm the result contains:
   - `preflight_requires_approval_before_download=true`
   - `dry_run_does_not_write_cache=true`
   - `live_recovery_writes_cache_and_updates_registry=true`
   - `content_endpoint_reads_recovered_bytes=true`
   - `control_mutation_audit_recorded=true`
3. Confirm the final conclusion is
   `go_http_https_media_recovery_executor_live_verified`.
4. Rerun `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
   and confirm the current-turn artifact exposes the same smoke result.
