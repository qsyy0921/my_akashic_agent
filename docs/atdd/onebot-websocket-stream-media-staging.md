# OneBot WebSocket Stream Media Staging ATDD

## Goal

Verify that Go uses NapCat `upload_file_stream` for local QQ rich media on
WebSocket channels and that live failures, if any, are narrowed to platform
rich-media send.

## Preconditions

- `services/agent-runtime` is running on `127.0.0.1:8780`.
- NapCat account `1049511700` is logged in on `ws://127.0.0.1:3001`.
- `scripts/run-qq-group-live-smoke.ps1` can target group `27234224`.

## Acceptance Scenarios

1. Unit verification:
   - run `go test ./infrastructure/onebotdelivery`
   - expected: WebSocket image/file tests prove action order includes
     `upload_file_stream` before final send/upload actions.

2. Live group text verification:
   - run `.\scripts\run-qq-group-live-smoke.ps1 -GroupId 27234224`
   - expected: text case still returns non-empty `provider_message_id` and
     outbox `status=succeeded`.

3. Live group image/file verification:
   - use the same smoke output and NapCat logs
   - expected success case:
     - final rich-media sends succeed
   - expected blocked case:
     - runtime and NapCat logs show the final failure after stream staging, with
       platform-side `rich media transfer failed`
     - the result is recorded as a platform/session blocker, not a local-path
       or container-visibility bug.
