# TDD: Telegram Backend Verification

## Targeted Coverage

1. Repo verification entrypoint:
   - read-only script can summarize current Telegram backend state
2. Live evidence surface:
   - runtime-config token status
   - receiver-statuses Telegram presence
   - optional `getMe` when token is present
3. Regression guard:
   - future iterations can distinguish `token_missing` from `getme_failed`
     without rebuilding ad hoc commands

## Commands

- `.\scripts\verify-telegram-backend.ps1`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- No unit test is added for the PowerShell Telegram `getMe` branch in this
  slice because the authoritative evidence is the live runtime plus live token
  environment.
