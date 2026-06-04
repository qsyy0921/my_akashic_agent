# SPEC-141 OneBot WebSocket Stream Media Staging

## Problem

The local Go runtime talks to NapCat through OneBot WebSocket endpoints
(`ws://127.0.0.1:3001` / `3002`) running inside Docker containers.

For QQ rich media, the previous adapter behavior had two code-side problems:

1. local Windows file paths could not be read by the containerized NapCat
   process;
2. local-path fallback depended on `base64://...` direct sends, which did not
   prove whether the Go runtime had already reached NapCat's current
   recommended upload path.

This left live failures ambiguous: some were transport/path issues, others were
platform/session rich-media failures.

## Goal

For WebSocket-configured OneBot channels, local media files should be staged
through NapCat `upload_file_stream` before Go asks NapCat to send an image or
upload a file.

The result should let live smoke distinguish:

- Go adapter path issues;
- Docker path-visibility issues;
- real NapCat / QQ rich-media platform failures.

## Non-Goals

- Do not claim QQ rich media is solved if the platform still returns
  `rich media transfer failed`.
- Do not move QQ sending ownership back into Python.
- Do not implement a Telegram media pipeline here.
- Do not add automatic retries or session repair logic in this slice.

## Required Behavior

When `EndpointConfig.WebSocketURL` is configured and a delivery step references
local media:

1. the adapter resolves the local filesystem path, including `file://` URLs;
2. the adapter reads the local file bytes;
3. the adapter uploads the bytes with `upload_file_stream`;
4. the adapter requires a non-empty returned `data.file_path`;
5. the adapter uses that returned `file_path` in the final `send_group_msg`,
   `send_private_msg`, `upload_group_file`, or `upload_private_file` action.

For HTTP/HTTPS, `base64://`, and `data:` media references, existing passthrough
behavior remains allowed.

## Verification

- unit tests must prove WebSocket image/file sends stage local media with
  `upload_file_stream` before final delivery actions;
- real group smoke must be rerun after the change;
- if rich media still fails live, the evidence must show that the final failure
  happens after stream staging, so the remaining blocker is platform/session
  level rather than Go path construction.
