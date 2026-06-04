# Spec 149: native private rich-media route verification

## Context

QQ rich-media verification is already split by account:

- image fails natively on both QQ accounts in group chats
- file succeeds on the second account in group chats
- file fails on the first account in group chats

That still leaves an important ambiguity: whether the first-account file
failure is account-wide, or only affects the group route.

## Goal

Verify native NapCat and Akashic delivery behavior for private rich-media so
the remaining blocker can be expressed as route-specific, not just account-
specific.

## Requirements

1. The native NapCat rich-media smoke script must support:
   - `conversation_type=group`
   - `conversation_type=private`
2. In private mode, the script must:
   - send image through `send_private_msg`
   - send file through `upload_private_file`
3. Run native private verification for:
   - `1049511700 -> 2365524513`
   - `2365524513 -> 1049511700`
4. Run Akashic manual private dispatch for the first account so the Go adapter
   result can be compared with native private behavior.

## Acceptance

- Native private image fails on both accounts with the same
  `rich media transfer failed` platform error.
- Native private file succeeds on both accounts.
- Akashic manual first-account private file succeeds.
- Akashic manual first-account private image fails with the same platform error.
- The remaining blocker is updated from vague “first-account file failure” to
  the narrower statement:
  - image fails across account and route boundaries
  - first-account file failure is currently group-route-specific

## Non-Goals

- Do not claim QQ image is fixable in Akashic if native private also fails.
- Do not expand default Go outbox ownership in this slice.
- Do not change Telegram behavior.
