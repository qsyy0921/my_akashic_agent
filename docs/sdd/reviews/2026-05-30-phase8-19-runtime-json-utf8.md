# Review: Phase 8.19 Runtime JSON UTF-8

Spec:
- `docs/sdd/specs/agent-gateway/001-message-envelope.md`

Implementation summary:
- Updated the Go runtime shared JSON writer to emit
  `Content-Type: application/json; charset=utf-8`.
- Added an HTTP handler regression test using `/healthz`.

Tests run:
- `go test ./...`

Findings:
- Windows PowerShell 5.1 may decode UTF-8 JSON responses incorrectly when the
  response only declares `application/json`.
- Runtime logs and Python I/O can be UTF-8 while API clients still display
  mojibake if the HTTP response omits charset.

Decision:
- Accept. The change is centralized in `writeJSON`, so all Go runtime JSON APIs
  now share the same explicit UTF-8 contract.

Follow-ups:
- If dashboard proxy endpoints add custom JSON writers, apply the same charset
  rule there as well.
