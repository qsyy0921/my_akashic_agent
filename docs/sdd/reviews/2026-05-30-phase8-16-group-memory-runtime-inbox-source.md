# Review: Group Memory Runtime Inbox Source

Spec:
- `docs/sdd/specs/agent-gateway/009-inbox-raw-message-store.md`
- `docs/sdd/specs/group-message-memory/001-processing-pipeline.md`
- `docs/sdd/specs/agent-architecture/010-python-ddd-suitability.md`

Implementation summary:
- Extended Go `InboxEventFilter` with `after_seq` and `order` so replay callers
  can fetch oldest-first batches without skipping messages.
- Updated memory and file-backed Go inbox repositories to honor the new cursor
  filter based on session `metadata.seq`.
- Added a Python group-memory message source port and preserved the existing
  session-store source as fallback.
- Added `AgentRuntimeInboxGroupMessageSource`, an adapter from Go `/v1/inbox`
  to the row shape consumed by `GroupMemoryService`.
- Wired agent-runtime knowledge worker startup to use the Go inbox source while
  leaving direct group-memory loop fallback unchanged when runtime is disabled.
- Documented why Go uses tactical DDD and why Python should keep
  ports/adapters plus AI-pipeline boundaries instead of full DDD.

Tests run:
- `python -m compileall group_memory integrations bootstrap`
- `uv run pytest tests\test_group_memory.py tests\test_agent_runtime_inbox_source.py tests\test_agent_gateway_client.py tests\test_bootstrap_wiring_p2.py -q --basetemp .tmp\pytest-runtime-inbox-source-full`
- `go test ./...` from `services/agent-runtime`

Findings:
- Cursor-safe replay requires source events to carry `metadata.seq`. Shadow
  mirroring of observed session messages already provides this metadata.
- Events without a stable sequence are intentionally ignored by the Python inbox
  source to avoid corrupting group-memory cursors.

Decision:
- Proceed with this observe-only cutover. Go owns durable inbox and cursor-safe
  replay; Python remains responsible for memory extraction and RAG logic.

Follow-ups:
- Add a Go-owned leased projection for long-running historical backfills.
- Attach media asset ids to extracted group-memory evidence.
- Consider moving RAGFlow ingest source reading to Go inbox after the current
  group-memory source cutover is stable.
