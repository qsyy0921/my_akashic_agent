# ATDD: Telegram Backend Verification

## Goal

Verify from current runtime and current environment whether Telegram backend is
blocked by token absence or by a later API/runtime failure.

## Preconditions

- `agent-runtime` is healthy on `127.0.0.1:8780`
- current repo has access to `scripts/verify-telegram-backend.ps1`

## Acceptance Checks

1. Run:
   - `.\scripts\verify-telegram-backend.ps1`
2. Confirm the JSON reports current runtime delivery state from
   `/v1/runtime-config`.
3. Confirm the JSON reports current receiver state from
   `/v1/receiver-statuses`.
4. In the current live runtime, confirm:
   - `runtime_reports_token_configured=false`
   - `local_env_token_present=false`
   - `telegram_receiver_present=false`
   - `backend_state=token_missing`
5. When a real token is later provided, rerun the same script and confirm:
   - `getme_attempted=true`
   - either `getme_ok=true`, or a concrete API/runtime failure is returned

## Failure Signals

- script cannot read runtime endpoints
- token is present but `getMe` is not attempted
- script claims ready without token or `getMe`
- script hides the difference between token absence and API failure
