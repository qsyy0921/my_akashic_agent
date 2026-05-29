# SPEC-009: Go Migration Implementation Design

## Status

Draft for review before further migration.

## Goal

Move all backend-infrastructure responsibilities that benefit from Go into
DDD + hexagonal Go services, while keeping AI/model-facing behavior in Python.

This is not a rewrite. Each step must leave the running project usable and must
have a rollback path.

## Non-Goals

- Do not move LLM inference, prompt engineering, RAG ranking, or memory
  extraction algorithms to Go.
- Do not cut over QQ/Telegram production delivery without shadow verification.
- Do not make observe-only QQ groups reply automatically.
- Do not keep growing Python channel adapters with new infrastructure
  responsibilities.

## Ownership Matrix

| Capability | Final Owner | Reason | First Migration Artifact |
| --- | --- | --- | --- |
| QQ/TG account registry | Go | deterministic account/session ownership | `ChannelAccount` aggregate |
| Inbound normalization | Go | one schema for all platforms | `MessageEnvelope` |
| Human/self/peer-bot classification | Go | testable without LLM | `ProvenanceClassifier` |
| Loop guard | Go | durable nonce/hop/send-ledger rules | `LoopGuard` |
| Routing bindings | Go | account-aware route safety | `RoutingBinding` |
| Durable inbox | Go | replay, retry, idempotency | `InboxEvent` repository |
| Outbox state | Go | retries, receipts, dead letter | `OutboxDelivery` |
| Platform dispatch adapters | Go | SDK isolation and retry ownership | `DeliveryAdapter` port |
| Media/file registry | Go | stable asset ids and dashboard links | `MediaAsset` |
| Image/RAG/memory job lifecycle | Go | long-running job recovery | `AgentJob` |
| Scheduler infrastructure | Go | leases, ticks, retries | `Schedule` and `Tick` |
| Audit/event store | Go | replayable typed history | `AuditEvent` |
| Dashboard query APIs | Go first, Python fallback | typed state APIs | `GatewayQueryService` |
| LLM reasoning | Python | provider and prompt iteration | `agent/runtime` |
| Tool execution | Python | ecosystem and local effects | `tool_runtime` |
| Memory/RAG algorithms | Python | embeddings, eval, summarization | `memory_runtime` workers |

## Target Go Bounded Contexts

```text
services/message-gateway
├── api
├── app
│   ├── command
│   ├── query
│   ├── port
│   │   ├── in
│   │   └── out
│   └── service
├── domain
│   ├── model
│   └── service
├── infrastructure
│   ├── memory
│   ├── auditjsonl
│   ├── queue
│   ├── napcat
│   ├── telegram
│   ├── media
│   └── persistence
├── trigger
│   ├── http
│   └── worker
└── types
```

## Application Ports

Inbound ports:

- `MessageIngestor`
- `ShadowMessageIngestor`
- `OutboxManager`
- `MediaAssetManager`
- `JobManager`
- `RoutingManager`
- `SchedulerManager`

Outbound ports:

- `MessageEventBus`
- `OutboxRepository`
- `OutboxQueue`
- `DeliveryAdapter`
- `MediaRepository`
- `AssetDownloader`
- `JobRepository`
- `JobQueue`
- `AuditLog`
- `RoutingRepository`
- `Clock`

## Migration Phases

### Phase A: Design And Contract Lock

Deliverables:

- SDD ownership matrix.
- Contract fixtures for messages, assets, jobs, citations, outbox, and routing.
- Architecture tests enforcing Go layer dependencies.

Acceptance:

- Go and Python both validate shared fixtures.
- No production runtime behavior changes.

### Phase B: Shadow Control Plane

Deliverables:

- Shadow inbound mirror.
- Audit JSONL.
- Dashboard shadow audit view.
- Outbox control-plane state.

Acceptance:

- Go receives mirrored QQ/TG events.
- `/v1/outbox` can track delivery state without sending.
- Existing Python direct delivery still works.

### Phase C: Media/File Registry

Deliverables:

- `MediaAsset` aggregate.
- Go asset registration endpoint.
- Local/signed dashboard asset URL.
- Python receives asset ids instead of raw CQ URLs.

Acceptance:

- QQ image/file metadata is registered by account id and group/private route.
- Dashboard links resolve through controlled routes.
- Observe-only groups still do not reply.

### Phase D: Job Queue

Deliverables:

- Unified `AgentJob` for image, RAG ingest, and memory extraction.
- Pending/running/succeeded/failed/dead-letter states.
- Python worker client for job lease and result update.

Acceptance:

- Multiple image jobs from different users can run concurrently.
- Failed jobs retry and then dead-letter.
- Group memory/RAG jobs cite source messages/assets.

### Phase E: Inbound Routing Cutover

Deliverables:

- Go owns route binding and normalized allowed inbox.
- Python consumes only `AgentInboundEvent`.
- Bot-to-bot messages require explicit protocol.

Acceptance:

- Two QQ accounts receive concurrently.
- Peer bot messages without protocol are observed only.
- Replies use the receiving account id.

### Phase F: Outbound Delivery Cutover

Deliverables:

- Go `DeliveryAdapter` implementations for NapCat and Telegram.
- Python returns `AgentDecision` to Go.
- Go dispatches text/files/images and records receipts.

Acceptance:

- Python does not call platform SDK send methods for migrated channels.
- Failed sends appear in outbox and dashboard.
- Rollback can restore Python compatibility send path.

### Phase G: Scheduler And Dashboard Consolidation

Deliverables:

- Go scheduler emits durable ticks and leases.
- Dashboard reads messages/assets/jobs/audit from Go APIs.
- Python keeps memory/RAG admin APIs until mirrored summaries exist.

Acceptance:

- Scheduler restart does not lose due ticks.
- Dashboard no longer scrapes Python session internals for migrated resources.

## Cutover Rules

- Every phase requires:
  - SDD spec or spec update;
  - Go unit tests;
  - Python compatibility tests when a Python boundary changes;
  - review note under `docs/sdd/reviews`;
  - runtime smoke test after service restart.
- A production adapter cutover requires:
  - feature flag;
  - rollback flag;
  - shadow comparison evidence;
  - route/account tests for both QQ accounts.

## Review Checklist

- Does the change move infrastructure to Go without moving AI logic?
- Does the domain layer stay free of app/infrastructure imports?
- Is account id present in every route, asset, job, and delivery?
- Are observe-only groups protected from outbound replies?
- Can failed sends/jobs be inspected and retried?
- Can the change be rolled back without re-login or data loss?
