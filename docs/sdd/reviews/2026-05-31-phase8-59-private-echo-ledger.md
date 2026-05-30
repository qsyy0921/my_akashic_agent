# Review: Phase 8.59 Private Echo Ledger

## Scope

- Added a Go-owned read-only private echo check:
  `GET /v1/send-ledger/private-echo`.
- Centralized compatibility echo rules for text and empty-text image/file/forward
  sends through the Go send ledger.
- Updated the Python QQ compatibility channel to prefer the new Go endpoint and
  retain the older `recently_sent` query as a fallback for older runtimes.

## Design

- Domain constants define the shared outbound markers: `[图片]`, `[文件]`, and
  `[转发消息]`.
- `SendLedgerService.CheckPrivateEcho` chooses text first, then marker-based
  attachment checks, and returns an explainable reason.
- The HTTP route is read-only; it does not record sends, mutate nonce state, or
  publish inbound messages.
- The compatibility bridge moves deterministic echo classification toward Go
  while preserving Python fallback behavior.

## Verification

- `go test ./app/service ./trigger/http -run "TestSendLedger.*Echo|TestSendLedgerEndpoint" -count=1 -v`
- `go test ./...`
- `uv run pytest tests/test_agent_gateway_client.py tests/test_channel_clients.py -k "private_echo or runtime_send_ledger_echo" -q`

## Remaining Risk

- Telegram currently only records sends into the Go send ledger; it does not yet
  query the new private echo endpoint because Telegram echo behavior differs
  from QQ/NapCat private-message callbacks.
