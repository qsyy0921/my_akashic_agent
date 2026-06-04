# TDD: Media Asset Content Recovery HTTP Executor Live Smoke

## Tests

1. Add focused unit tests for
   `scripts/verify_media_asset_content_recovery_live_smoke.py` covering:
   - live-verified conclusion
   - failure conclusion
   - error conclusion
2. Extend `tests/test_verify_go_migration_goal_script.py` to assert:
   - the new PS1 wrapper path is wired
   - the new smoke verifier is invoked
   - the smoke result is surfaced in unified artifact fields
3. Rerun the focused verifier tests.
4. Run the PowerShell smoke and the unified goal verifier end to end.
