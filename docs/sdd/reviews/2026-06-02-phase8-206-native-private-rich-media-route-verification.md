# Review: native private rich-media route verification

## Scope

- `docs/sdd/specs/agent-gateway/149-native-private-rich-media-route-verification.md`
- `scripts/run-napcat-native-rich-media-smoke.ps1`

## What changed

- Extended the native NapCat smoke entrypoint so it can target both group and
  private conversations.
- Re-ran native private rich-media verification on both QQ accounts.
- Re-ran Akashic manual private image/file dispatch on the first QQ account to
  compare adapter behavior with native private results.

## Verification

- `scripts/run-napcat-native-rich-media-smoke.ps1 -ConversationType private -ChatId 2365524513`
- `scripts/run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -ConversationType private -ChatId 1049511700`
- Manual Akashic private dispatch:
  - first-account file
  - first-account image

## Result

- Native private image still fails on both accounts with
  `rich media transfer failed`.
- Native private file succeeds on both accounts.
- Akashic first-account private file succeeds.
- Akashic first-account private image fails with the same platform error.
- Accept. The remaining blocker is now narrower:
  - image failure is cross-account and cross-route
  - first-account file failure is currently group-route-specific, not
    account-wide
