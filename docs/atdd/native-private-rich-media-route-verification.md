# ATDD: Native Private Rich-media Route Verification

## Goal

Verify whether QQ rich-media failure is group-only or also appears in private
chat, and confirm that Akashic matches native private behavior.

## Preconditions

- NapCat containers are connected on:
  - `ws://127.0.0.1:3001`
  - `ws://127.0.0.1:3002`
- Two QQ accounts are logged in:
  - `1049511700`
  - `2365524513`
- `agent-runtime` is healthy on `127.0.0.1:8780`

## Acceptance Checks

1. Run:
   - `scripts/run-napcat-native-rich-media-smoke.ps1 -ConversationType private -ChatId 2365524513`
   - `scripts/run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -ConversationType private -ChatId 1049511700`
2. Confirm for both runs:
   - `image_upload.retcode=0`
   - `image_send.retcode=1200`
   - `image_send.message` contains `rich media transfer failed`
   - `file_upload.retcode=0`
   - `file_send.retcode=0`
3. Manually dispatch first-account private file through Akashic and confirm:
   - `dispatch.results[0].status=sent`
   - final outbox state is `succeeded`
4. Manually dispatch first-account private image through Akashic and confirm:
   - platform error contains `rich media transfer failed`
   - final outbox state is not marked as succeeded

## Failure Signals

- native private image succeeds on one side while Akashic private image fails
- native private file fails on either account
- Akashic private file fails while native private file succeeds
