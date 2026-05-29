# SPEC-008: Go Migration Scope

## Status

Accepted as the full migration target. Individual cutovers still require their
own review record and tests.

## Context

The project should not use Go only for the visible message gateway. Go should
own backend infrastructure where deterministic state, concurrency, retries,
durability, and platform protocol boundaries matter. Python should stay focused
on agent intelligence and model-facing behavior.

## Go-Owned Targets

| Area | Target Go Owner | Migration Priority | Notes |
| --- | --- | --- | --- |
| Message gateway | `services/message-gateway` | P0 | normalize QQ/Telegram/Feishu/WeChat events |
| Account registry | `domain/model/ChannelAccount` | P0 | one identity per QQ/TG bot account |
| Inbound routing | `RoutingService` | P0 | route by platform + account + conversation |
| Loop guard | `LoopGuard` | P0 | self echo, peer bot protocol, nonce, hop budget |
| Shadow audit | `AuditService` | P0 | already started with JSONL shadow events |
| Outbox state | `OutboxService` | P0 | queued/dispatching/succeeded/failed/dead-letter |
| Platform delivery adapters | `infrastructure/napcat`, `infrastructure/telegram` | P1 | only after shadow and outbox state pass |
| Media/file registry | `MediaService` | P1 | asset ids, retention, local/signed URLs |
| Image job control plane | `JobService` | P1 | Python executes generation; Go owns status |
| RAG ingest jobs | `JobService` | P1 | Python embeds/ranks; Go owns lifecycle/retry |
| Group memory extract jobs | `JobService` | P1 | observe-only invariant enforced before Python |
| Durable queues | `infrastructure/queue` | P1 | NATS/Redis Streams adapter behind app ports |
| Scheduler infrastructure | `SchedulerService` | P2 | Go emits ticks; Python decides LLM content |
| Dashboard query API | `GatewayQueryService` | P2 | typed messages/assets/jobs/audit resources |
| Metrics/tracing | `infrastructure/telemetry` | P2 | OpenTelemetry after boundary stabilizes |

## Python-Owned Targets

| Area | Why Python |
| --- | --- |
| Provider/model routing | MiMo, vision model, embedding, fallback behavior changes often |
| Prompt/context pipeline | LLM-quality logic and token budgeting |
| Tool execution | rich Python ecosystem and local scripts |
| Memory extraction | embeddings, summarization, dedupe, RAG libraries |
| RAG ranking/evaluation | ML/eval stack and experiments |
| Image generation worker | current ChatGPT proxy/browser-specific implementation |
| Agent turn state machine | model recovery, tool loops, and final decision logic |

## Migration Order

1. Shadow audit and contracts.
2. Inbound routing and loop guard.
3. Outbox state and delivery receipts.
4. Media/file registry.
5. Job queue for image/RAG/memory workers.
6. Platform delivery adapters.
7. Python channel thinning.
8. Dashboard query consolidation.
9. Scheduler infrastructure.

## Safety Rules

- Do not move LLM calls into Go routing or delivery.
- Do not cut over platform delivery without shadow and rollback.
- Observe-only groups must never emit group-visible replies.
- Every migrated boundary needs Go unit tests and Python compatibility tests or
  fixture validation.
- Existing Python platform files should only receive bug fixes until their Go
  replacement is ready.
