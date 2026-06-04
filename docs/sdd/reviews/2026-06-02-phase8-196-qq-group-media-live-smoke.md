# Review: QQ group media live smoke

Spec:

- `docs/sdd/specs/agent-gateway/139-qq-group-media-live-smoke-runbook.md`

Implementation summary:

- Added `scripts/run-qq-group-live-smoke.ps1` to run one real QQ group text,
  image, and file smoke against an enabled observe-only group target.
- Hardened the Go OneBot WebSocket adapter so unrelated non-echo frames with a
  non-string `message` payload no longer abort media sends before the echoed
  action response arrives.
- Updated runtime docs and SDD/ATDD/TDD/governance docs to record the real live
  smoke outcomes and the remaining cutover blocker.

Tests run:

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./infrastructure/onebotdelivery`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `.\scripts\run-qq-group-live-smoke.ps1 -GroupId 27234224`
- `POST http://127.0.0.1:8780/v1/outbound-cutover/readiness`
- `POST http://127.0.0.1:8780/v1/outbound-cutover/plan`
- `GET http://127.0.0.1:8780/v1/queue-backend`

Findings:

- The adapter bug was real: NapCat media-related non-echo frames can carry an
  array `message` field, which previously caused JSON decode failure before the
  echoed response was read.
- After the adapter fix and runtime restart, group text smoke succeeds through
  Go `delivery-dispatch/send`.
- Group image and file sends now fail at the real platform boundary with
  `rich media transfer failed`, which is a stronger and more actionable blocker
  than the prior adapter parse failure.
- Cutover remains blocked because the execution owner is still
  `go_state_store_api`; the current plan recommends `go_local_outbox_worker`,
  not NATS external lease, on this local provider setup.

Decision:

- Accepted.

Follow-ups:

- Investigate NapCat / QQ rich-media upload prerequisites for the logged-in
  `1049511700` session.
- Re-run `scripts/run-qq-group-live-smoke.ps1` after rich-media remediation.
- Only after group image/file smoke passes should the local outbox worker be
  considered for enablement.
