# Message Gateway Specs

## Scope

The Go message gateway owns platform messaging infrastructure:

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
- Agent architecture boundary specs live under `../agent-architecture/`.
