# Spec 150: account-conversation-kind outbox cutover gate

## Context

The current Go local outbox worker already owns real QQ delivery for:

- global `text`
- second-account `2365524513` `file`

Live verification has narrowed the remaining QQ rich-media surface:

- `image` is still failing natively across both QQ accounts and across both
  `group` and `private` routes with `rich media transfer failed`
- first-account `1049511700` `file` is not globally broken:
  - `private` succeeds natively and through Akashic
  - `group` still fails

Account-scoped gating is no longer precise enough. Keeping first-account
`file` fully blocked prevents Go from owning a route that is already proven
safe (`1049511700/private/file`), but globally opening first-account `file`
would incorrectly expose the known-bad `1049511700/group/file` route.

## Goal

Allow Go local outbox execution to gate supported delivery kinds by account,
conversation type, and kind, so partial cutover can safely expand to:

- all text
- second-account file
- first-account private file

without claiming first-account group file or any image readiness.

## Requirements

1. Runtime configuration must accept an account+conversation gate:
   - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE`
   - format: `account/private=kind|kind,account/group=kind`
2. Lease filtering must prefer the most specific gate in this order:
   - account + conversation type
   - account
   - global
3. Runtime diagnostics must expose the new gate in:
   - `/v1/runtime-config`
   - `/v1/queue-backend`
   - `/v1/runtime-workers`
4. `outbox_execution_scope` must reflect the new mode as
   `account_conversation_kind_gated`.
5. When local outbox worker runs with:
   - global `text`
   - account override `2365524513=text|file`
   - route override `1049511700/private=text|file`
   then live QQ outbox behavior must be:
   - first-account private file auto-succeeds through Go local worker
   - first-account group file remains queued with `attempts=0`
   - second-account group file still auto-succeeds

## Non-Goals

- Do not claim QQ image readiness.
- Do not infer route capability automatically from prior history.
- Do not expand Telegram cutover in this slice.
- Do not change Python AI worker ownership outside the existing compatibility
  outbox backoff behavior.

## Acceptance

- `/v1/runtime-config` reports:
  - `workers.outbox_delivery_allowed_kinds=["text"]`
  - `workers.outbox_delivery_allowed_kinds_by_account.2365524513=["text","file"]`
  - `workers.outbox_delivery_allowed_kinds_by_account_conversation_type.1049511700.private=["text","file"]`
- `/v1/queue-backend` reports:
  - `outbox_execution_owner=go_local_outbox_worker`
  - `outbox_execution_scope=account_conversation_kind_gated`
  - the same route-scoped gate
- `/v1/runtime-workers` reports
  `allowed_step_kinds_by_account_conversation_type=1049511700/private=text|file`
- A real first-account private file outbox event reaches `succeeded` without
  manual dispatch or manual mark-succeeded.
- A real first-account group file outbox event remains `queued` with
  `attempts=0`.
- A real second-account group file outbox event still reaches `succeeded`.
