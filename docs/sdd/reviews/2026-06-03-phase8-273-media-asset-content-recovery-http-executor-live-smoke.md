# Review: Media Asset Content Recovery HTTP Executor Live Smoke

## What changed

- Added `scripts/verify_media_asset_content_recovery_live_smoke.py` and
  `scripts/verify-media-asset-content-recovery-live-smoke.ps1`.
- Added focused regression tests in
  `tests/test_verify_media_asset_content_recovery_live_smoke.py`.
- Wired the new smoke into
  `scripts/verify-go-migration-goal.ps1` and its focused test.

## Why

The existing media recovery boundary verifier only proved the current runtime
classification and dashboard/read-model visibility. It did not prove the Go
HTTP/HTTPS recovery executor itself on the current turn.

## Evidence

- `uv run pytest tests/test_verify_media_asset_content_recovery_live_smoke.py tests/test_verify_go_migration_goal_script.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-media-asset-content-recovery-live-smoke.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
- The unified artifact now contains
  `media_recovery_executor_smoke.conclusion.status=live_verified`.
