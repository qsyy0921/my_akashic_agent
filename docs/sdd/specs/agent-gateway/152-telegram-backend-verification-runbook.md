# Spec 152: telegram backend verification runbook

## Context

Telegram backend cutover is still blocked by missing token-backed verification.
Current runtime diagnostics already expose:

- whether Telegram token is configured
- whether any Telegram receiver is present

But each iteration still has to reassemble the same evidence manually before
deciding whether Telegram is blocked by token absence or by a later runtime
failure.

## Goal

Provide a repo-owned read-only verification entrypoint that checks Telegram
backend readiness from current runtime state and, when a token is present, runs
`getMe` against the configured Telegram API endpoint.

## Requirements

1. Add a repo-owned script:
   - `scripts/verify-telegram-backend.ps1`
2. The script must read live runtime state from:
   - `/v1/runtime-config`
   - `/v1/receiver-statuses`
3. The script must detect token presence from:
   - explicit `-TelegramBotToken`
   - `TELEGRAM_BOT_TOKEN`
   - `AKASHIC_TELEGRAM_BOT_TOKEN`
4. When a token is present, the script must call:
   - `<telegram_api_base>/bot<TOKEN>/getMe`
5. The script must emit machine-readable JSON that makes these checks explicit:
   - runtime reports token configured or not
   - local env token is present or not
   - Telegram receiver exists or not
   - `getMe` attempted or not
   - `getMe` succeeded or not
   - backend state summary such as `token_missing`, `getme_ok`, `getme_failed`
6. The script must remain read-only:
   - no receiver start/stop
   - no message send
   - no config mutation

## Non-Goals

- Do not fake Telegram readiness when no token is present.
- Do not create or modify Telegram bot configuration.
- Do not add a receiver boot path in this slice.

## Acceptance

- Running `.\scripts\verify-telegram-backend.ps1` returns JSON.
- In the current live runtime, the JSON proves:
  - `runtime_reports_token_configured=false`
  - `local_env_token_present=false`
  - `telegram_receiver_present=false`
  - `getme_attempted=false`
  - `backend_state=token_missing`
- When a valid token is provided later, the same script becomes the first
  verification entrypoint for `getMe` and receiver follow-up checks.
