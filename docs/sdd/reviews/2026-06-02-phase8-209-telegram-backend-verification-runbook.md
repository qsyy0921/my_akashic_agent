# Phase 8.209 Review: telegram backend verification runbook

## What changed

- Added a repo-owned read-only verification script:
  - `scripts/verify-telegram-backend.ps1`
- The script checks live runtime Telegram status and, when a token is present,
  attempts Telegram `getMe`.

## What was verified

- `.\scripts\verify-telegram-backend.ps1` ran successfully against the current
  runtime and returned JSON.
- The current runtime proved:
  - `telegram_token_configured=false`
  - no Telegram receiver in `/v1/receiver-statuses`
  - no local env token present
  - `getMe` was not attempted
  - backend state is `token_missing`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
  passed.

## Outcome

Telegram backend is now explicitly verified as blocked by token absence, not by
an already-configured receiver or by a hidden `getMe` failure. The goal remains
active because no real token-backed smoke has run yet, and QQ rich-media image
/ first-account group-file blockers still exist.
