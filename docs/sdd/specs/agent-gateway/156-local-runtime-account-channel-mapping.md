# Spec 156: Local Runtime Account Channel Mapping

## Why

Repo-local `scripts/start-agent-runtime.ps1` is the default bring-up path for
this migrated workspace. When the script enables the Go local outbox worker for
multi-account QQ smoke, it must also inject a deterministic
`AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` mapping. Otherwise the worker may route a
second-account delivery through the primary `qq` alias and create a false
cutover regression that does not reproduce in native NapCat parity checks.

## Requirements

1. `scripts/start-agent-runtime.ps1` must export
   `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT=1049511700=qq_1049511700,2365524513=qq_2365524513`
   by default for the local migrated workspace.
2. After repo-local bring-up with the Go local outbox worker enabled,
   `/v1/runtime-workers` must show per-account channel mapping attributes for
   the outbox worker.
3. Repo-owned live smoke must again show:
   - `first_account_private_file` succeeds
   - `second_account_group_file` succeeds
   - `first_account_group_file` remains gated
   - `first_account_private_image` remains gated
4. The unified goal verifier must no longer report a false
   `outbox_scope_case_failed_second_account_group_file` blocker after the local
   runtime is restarted with the fixed launcher.

## Non-Goals

- Changing platform/session-level QQ rich-media behavior.
- Expanding Go owner beyond the current account/conversation/kind gate.
- Solving Telegram token absence.
