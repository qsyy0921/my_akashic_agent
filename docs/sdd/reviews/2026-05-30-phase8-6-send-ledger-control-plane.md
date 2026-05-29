# Review: Phase 8.6 Go Send Ledger Control Plane

Spec: `docs/sdd/specs/agent-architecture/009-go-migration-implementation-design.md`

Implementation summary:

- Added a Go-owned `SendLedgerManager` inbound port with `Record`,
  `RecentlySent`, and `List` use cases.
- Added `/v1/send-ledger/records` and `/v1/send-ledger/recent` HTTP endpoints so
  Python platform senders can record outbound text and query echo risk without
  owning the loop-guard state.
- Added a file-backed `sendledgerstore` adapter selected by
  `AKASHIC_SEND_LEDGER_DSN` or `AKASHIC_SEND_LEDGER_PATH`.
- Updated runtime wiring so inbound loop guard and outbound send recording use
  the same `SendLedger` repository instead of unrelated in-memory stores.

Tests run:

- `gofmt -w api app cmd domain infrastructure trigger types`
- `go test ./...`

Findings:

- This slice does not change QQ/Telegram delivery behavior. It only exposes the
  shared send ledger that the Python compatibility layer can call next.
- The domain model remains intentionally small: `from_bot_id`,
  `conversation_id`, `content_hash`, and `timestamp`. Platform, route type, and
  richer delivery metadata stay in outbox/message records unless a later use
  case needs them in the echo ledger itself.
- The ledger is append-only for now. Pruning and compaction should be added
  before using a long-lived high-volume JSON file in production.

Decision:

Accepted for the current Go migration. It moves a deterministic infrastructure
concern from Python-local process memory into the Go runtime boundary without
moving AI reasoning or platform-specific SDK dispatch yet.

Follow-ups:

- Wire `infra/channels/qq_channel.py` and Telegram send paths to call
  `/v1/send-ledger/records` after successful compatibility sends.
- Use `/v1/send-ledger/recent` in Python-only fallback guards until outbound
  delivery fully cuts over to Go.
- Add dashboard read-only send ledger diagnostics after Python senders are
  calling the new API.
