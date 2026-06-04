# OneBot WebSocket Stream Media Staging TDD

## Scope

Go adapter behavior for local QQ image/file sends over OneBot WebSocket.

## Required Tests

1. WebSocket image send with local file:
   - first action: `upload_file_stream` chunk
   - second action: `upload_file_stream` complete
   - final action: `send_group_msg`
   - final image segment must use returned `data.file_path`

2. WebSocket file upload with local file plus optional caption:
   - optional text message still sends first
   - local file staging still uses `upload_file_stream`
   - final upload action must use returned `data.file_path`

3. Existing HTTP behavior:
   - HTTP image/file tests continue to pass with current `base64://` fallback

## Non-Test Goals

- Do not fake live QQ success in unit tests.
- Do not add Telegram cases in this slice.
