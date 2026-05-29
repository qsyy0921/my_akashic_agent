# Review: Phase 8.7 Python Send Ledger Bridge

Spec: `docs/sdd/specs/agent-architecture/009-go-migration-implementation-design.md`

Implementation summary:

- Extended the Python `AgentRuntimeClient` compatibility client with
  `record_send` and `recently_sent` methods for the Go
  `/v1/send-ledger/*` contract.
- Wired channel bootstrap so Telegram and QQ channels receive one shared
  runtime client when `integrations.agent_runtime.enabled=true`.
- Updated QQ sends to record successful text/file/image/forward sends in the Go
  ledger. Private QQ inbound handling also checks the Go ledger for recent echo
  messages before allowing peer-bot trigger handling.
- Updated Telegram text/stream/file/image/final response sends to record
  successful sends in the Go ledger.

Tests run:

- `python -m pytest tests/test_agent_gateway_client.py tests/test_channel_clients.py tests/test_runtime_smoke.py`
- `go test ./...` under `services/agent-runtime`

Findings:

- The bridge is best-effort. Runtime ledger failures are logged and do not block
  platform delivery.
- QQ keeps the local short-lived private guard as a fallback while the Go ledger
  becomes the shared cross-process source of truth.
- Telegram does not yet have a stable platform bot id in config, so it records
  `from_bot_id` as the channel name unless the Telegram bot object exposes an
  id at runtime.

Decision:

Accepted as the next migration step. Python still owns SDK dispatch for this
phase, but send-loop state is now written to the Go infrastructure boundary.

Follow-ups:

- Add dashboard diagnostics for recent send records.
- Move QQ/TG SDK dispatch behind Go `DeliveryAdapter` after the send ledger has
  live evidence from both QQ accounts.
- Add pruning or compaction for long-lived send ledger JSON files.
