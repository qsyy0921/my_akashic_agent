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
- Agent architecture boundary specs live under `../agent-architecture/`.
