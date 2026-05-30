# Review: Phase 8.15 Go inbox raw message store

Spec:
- `docs/sdd/specs/agent-gateway/009-inbox-raw-message-store.md`
- `docs/sdd/specs/group-message-memory/001-processing-pipeline.md`

Implementation summary:
- Added Go domain aggregate `InboxEvent` for raw inbound/observed messages.
- Added `InboxEventRepository` and `InboxEventViewer` ports, plus app service
  view mapping.
- Wired `MessageIngestService` so normal and shadow ingest save raw inbox
  events before publishing agent work.
- Added file-backed `infrastructure/inboxstore` with idempotent duplicate
  handling and query filters.
- Added `/v1/inbox` and `/v1/inbox/{event_id}` HTTP query endpoints.
- Added `AKASHIC_INBOX_DSN` / `AKASHIC_INBOX_PATH` runtime configuration.

Tests run:
- Go `go test ./...` under `services/agent-runtime`.

Findings:
- This moves the raw group-message log toward Go ownership without changing
  Python QQ/NapCat adapters or group-memory extraction algorithms.
- Existing observe-only groups still do not emit outbound replies; the new path
  only records and queries normalized source messages.

Decision:
- Accepted as an observe-only migration slice. Go now has a typed raw inbox
  that can become the replay source for group memory and RAG citation work.

Follow-ups:
- Point Python group-memory extraction at `/v1/inbox` instead of Python session
  internals.
- Add cursor pagination before bulk historical replay.
- Link inbox event ids into RAG citations and dashboard source views.
