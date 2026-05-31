# Agent Gateway Specs

## Scope

The Go agent gateway owns platform messaging infrastructure:

- multi-account QQ connection;
- normalized message schema;
- provenance classification;
- signed bot-to-bot protocol;
- loop prevention;
- queue routing;
- audit events.

It does not own LLM reasoning, memory consolidation, RAG ranking, or tool
execution.

## Specs

- `001-message-envelope.md`: normalized inbound/outbound message schema.
- `002-provenance-and-bot-protocol.md`: human, self echo, peer bot, and signed
  bot-to-bot messages.
- `003-loop-guard.md`: TTL, budget, nonce, duplicate-content, and cooldown rules.
- `004-queue-and-routing.md`: NATS subjects, consumers, retries, and dead letters.
- `005-go-package-structure.md`: Go module package layout and dependency rules.
- `006-outbox-delivery-retry.md`: Go-owned outbound delivery state, retry, and
  dead-letter contract.
- `007-media-file-registry.md`: Go-owned QQ/TG media/file asset ids, metadata,
  retention, and safe dashboard access rules.
- `008-agent-job-orchestration.md`: Go-owned lifecycle, leasing, retry, and
  dead-letter control plane for image/RAG/memory jobs.
- `009-inbox-raw-message-store.md`: Go-owned raw observed/inbound message store
  for replay, dashboard queries, and group-memory source citations.
- `010-knowledge-checkpoints.md`: Go-owned knowledge ingestion cursors for
  RAGFlow and future memory/RAG workers.
- `012-agent-job-event-stream.md`: Go-owned durable lifecycle event stream for
  generic agent jobs.
- `013-rag-eval-jobs.md`: Go-owned `rag_eval` lifecycle with Python eval
  worker execution.
- `014-runtime-dashboard-overview.md`: read-only dashboard aggregation for
  runtime health, worker leases, stale jobs, dead letters, checkpoint lag, and
  job event stream.
- `015-delivery-dispatch-plan.md`: Go-owned outbound dispatch planning before
  Python compatibility workers perform platform sends.
- `016-telegram-delivery-adapter.md`: Go-owned Telegram HTTP Bot API sender
  behind the delivery dispatch boundary.
- `017-runtime-backed-message-push.md`: Python `message_push` enqueue path into
  Go `/v1/outbound` and outbox workers for supported channels.
- `018-onebot-napcat-delivery-adapter.md`: Go-owned OneBot/NapCat HTTP or
  WebSocket sender for env-gated QQ private/group text, image, and file
  dispatch.
- `019-rag-eval-dashboard.md`: read-only dashboard panel for Go-owned
  `rag_eval` quality gates, metrics, trend points, and per-question results.
- `020-outbox-event-stream.md`: Go-owned durable lifecycle event stream for
  outbox delivery state transitions.
- `021-external-queue-backend.md`: external queue backend selection and
  migration phases from local state-store leasing to NATS JetStream first,
  with bounded concurrent consumers and Redis/RabbitMQ kept behind
  provider-neutral ports.
- `022-send-ledger-metrics.md`: Go-owned recent-send ledger metrics for
  bot-to-bot loop guard observability.
- `023-observe-target-diagnostics.md`: Go-owned runtime diagnostics for
  configured observe-only QQ group targets synced from Python config.
- `024-receiver-status-diagnostics.md`: Go-owned runtime diagnostics for
  Python platform receiver lifecycle states, including QQ connected status and
  Telegram polling conflict suspension.
- `025-receiver-lease-control.md`: Go-owned receiver lease control for
  single-instance long-running platform polling loops.
- `026-observe-capture-diagnostics.md`: Go-owned read-only diagnostics for
  observe-only QQ group text/image/file capture and safe media content coverage.
- `027-receiver-status-heartbeat.md`: durable receiver status storage and
  Python QQ/Telegram heartbeat reporting for restart-safe connectivity
  diagnostics.
- `028-receiver-lease-persistence.md`: durable receiver lease storage and
  Telegram reacquire behavior after `agent-runtime` restarts.
- `029-observe-capture-activity-inference.md`: Go observe-capture diagnostics
  infer effective receiver connectivity from recent observe-only inbox events.
- `030-inbound-dedupe-runtime.md`: Go-owned TTL/persistent duplicate detection
  for inbound platform message ids, starting with Telegram receive paths.
- `031-agent-worker-status.md`: Go-owned liveness/status registry for Python
  AI workers while Python keeps model and tool execution.
- `032-proactive-anyaction-quota.md`: Go-owned deterministic AnyAction quota
  window state for proactive admission, with Python retaining probability logic.
- `033-proactive-seen-and-rejection-state.md`: Go-owned proactive source item
  seen dedupe and rejection cooldown state, with Python retaining semantic
  candidate extraction and tick logs.
- `034-proactive-retention-cleanup.md`: Go-owned TTL cleanup for proactive
  deterministic runtime state, while Python keeps SQLite fallback cleanup.
- `035-proactive-bg-context-global-mark.md`: Go-owned global mark for proactive
  background-context main-topic send timestamp.
- `036-scheduler-job-store.md`: Go-owned durable snapshot store for Python
  scheduler jobs, while Python keeps tick loop and AI execution.
- `037-scheduler-job-diagnostics.md`: Go-owned read-only scheduler snapshot
  diagnostics and runtime overview card.
- `038-scheduler-execution-lease.md`: Go-owned per-job scheduler execution
  lease/fencing, while Python keeps tick loop and job execution.
- `039-scheduler-job-crud.md`: Go-owned single scheduler job upsert/delete for
  tool-driven add/cancel, reducing snapshot overwrite risk.
- `040-scheduler-completion-mutation.md`: Go-owned lease-fenced scheduler
  completion mutation for recurring reschedule and one-shot delete.
- `041-scheduler-recovery-reconciliation.md`: Python startup scheduler recovery
  persists recurring misfire advancement and expired one-shot deletion through
  existing Go-owned scheduler CRUD.
- `042-proactive-drift-state.md`: Go-owned proactive drift skill state and
  recent-run summary, with Python retaining skill file scanning and AI/tool
  execution.
- `043-proactive-tick-log-state.md`: Go-owned proactive tick start/finish/step
  audit state, with Python retaining proactive behavior and SQLite dashboard
  mirror.
- `044-dashboard-proactive-tick-log-runtime-fallback.md`: Dashboard proactive
  tick log read fallback to Go runtime when SQLite mirror has no matching
  records, plus Go query pagination/sort filters.
- `045-agent-worker-status-lease-fencing.md`: Go-owned lease/fencing for
  Python AI worker status ownership, preventing active same-worker-id status
  overwrites from another process instance.
- Agent architecture boundary specs live under `../agent-architecture/`.
