# Review: Phase 8.58 Delivery Adapter Diagnostics

## Scope

- Added a Go-owned read-only delivery adapter diagnostics query path.
- Added `GET /v1/delivery-adapters` to expose configured Telegram and OneBot
  channel aliases, transport type, endpoint presence, redacted endpoint, and
  token presence.
- The endpoint does not perform live sends and does not expose token values.

## Design

- `cmd/agent-runtime` parses process-level adapter environment variables.
- `app/service` owns the stable diagnostics query service.
- `trigger/http` exposes the read-only route.
- Platform-specific send behavior remains in `infrastructure/telegramdelivery`
  and `infrastructure/onebotdelivery`; diagnostics only reports configuration.

## Verification

- `go test ./cmd/agent-runtime ./app/service ./trigger/http -run "TestDeliveryAdapter|TestDeliveryAdaptersEndpoint" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Remaining Risk

- This is configuration visibility, not a health check. It proves that runtime
  aliases and endpoints are parsed, but it deliberately avoids OneBot/Telegram
  network calls. Live send smoke still requires explicit operator approval.
