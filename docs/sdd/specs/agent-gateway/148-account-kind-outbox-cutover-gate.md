# Spec 148: account-kind outbox cutover gate

## Context

The current Go local outbox worker is already the real execution owner for QQ
text. Rich-media verification has been split more precisely:

- image is still failing natively across both QQ accounts with
  `rich media transfer failed`;
- file already works on account `2365524513`;
- file still fails on account `1049511700`.

Global `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS=text` is too narrow to let the
second account use working file delivery, but a global expansion to include
`file` would incorrectly expose the first account to a known session-specific
failure.

## Goal

Allow Go local outbox execution to gate supported delivery kinds by account, so
partial cutover can advance from `text_only` to account-scoped file support
without claiming full rich-media readiness.

## Requirements

1. Runtime configuration must accept an account-scoped allowed-kind override:
   - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT`
   - format: `account=kind|kind,account=kind`
2. Lease filtering must prefer account-scoped allowed kinds when a delivery's
   `channel.account_id` has an override.
3. Global `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS` remains the fallback for
   accounts without an override.
4. Runtime diagnostics must expose the new gate in:
   - `/v1/runtime-config`
   - `/v1/queue-backend`
   - `/v1/runtime-workers`
5. When local outbox worker runs with:
   - global `text`
   - account override `2365524513=text|file`
   then a real second-account QQ file outbox delivery must be auto-sent by Go,
   while a real first-account QQ file outbox delivery must stay queued.

## Non-Goals

- Do not claim QQ image readiness.
- Do not automatically infer capability from rich-media history.
- Do not add Telegram-specific cutover logic in this slice.
- Do not replace Python AI worker ownership outside the existing outbox backoff
  path.

## Acceptance

- `/v1/runtime-config` reports:
  - `workers.outbox_delivery_allowed_kinds=["text"]`
  - `workers.outbox_delivery_allowed_kinds_by_account.2365524513=["text","file"]`
- `/v1/queue-backend` reports:
  - `outbox_execution_owner=go_local_outbox_worker`
  - `outbox_execution_scope=account_kind_gated`
  - `outbox_allowed_kinds_by_account.2365524513=["text","file"]`
- `/v1/runtime-workers` reports the same override in
  `outbox_delivery_worker.attributes.allowed_step_kinds_by_account`
- A real second-account file outbox event reaches `succeeded` without manual
  dispatch or manual mark-succeeded.
- A real first-account file outbox event remains `queued` with `attempts=0`.
