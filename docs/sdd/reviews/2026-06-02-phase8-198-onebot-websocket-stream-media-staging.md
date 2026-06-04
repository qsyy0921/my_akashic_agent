# Phase 8.198 Review - OneBot WebSocket Stream Media Staging

## What Changed

- Go OneBot WebSocket delivery now stages local media through
  `upload_file_stream` before final image/file delivery actions.
- Local `file://` media references are normalized back to filesystem paths
  before stream upload or HTTP fallback.
- WebSocket unit tests now prove stream-upload action ordering for local image
  and file sends.

## Why

The repo runs NapCat inside Docker while Go runs on Windows host paths. Direct
local-path rich media could fail for at least two different reasons:

1. the container cannot see the Windows path;
2. the platform rich-media send itself fails.

By moving WebSocket local-media handling onto `upload_file_stream`, the Go side
now follows NapCat's current recommended staging path and removes path
visibility ambiguity from live smoke.

## Verification

- `go test ./infrastructure/onebotdelivery`
- `.\scripts\run-qq-group-live-smoke.ps1 -GroupId 27234224`

## Outcome

- QQ group text still succeeds.
- QQ group image/file still fail in the current live session, but the failure is
  now narrowed to the final NapCat/QQ rich-media send step:
  `rich media transfer failed`.
- This means the remaining blocker is platform/session-level rich media, not Go
  adapter local-path construction.
