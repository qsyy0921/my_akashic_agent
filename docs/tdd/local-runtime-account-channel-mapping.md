# TDD: Local Runtime Account Channel Mapping

## Coverage Targets

1. Repo-local launcher exports `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT`.
2. Live runtime reflects the account-channel mapping in worker diagnostics.
3. Live outbox scope smoke stays aligned with native NapCat parity for the
   second account group-file route.

## Tests

- Live restart through `scripts/start-agent-runtime.ps1`
- `.\scripts\verify-go-outbox-scope-live.ps1`
- `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
