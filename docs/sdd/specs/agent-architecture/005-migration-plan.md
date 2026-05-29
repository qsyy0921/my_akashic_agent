# SPEC-005: Migration Plan

## Status

Draft

## Context

The target architecture must be reached incrementally. A big-bang rewrite would
break the working Telegram/QQ/dashboard setup. Each phase below has a narrow
goal and an acceptance gate.

## Phase 0: Freeze Architecture Rules

Goal:

- Record Python/Go ownership.
- Add dependency tests for Go DDD layers.
- Stop adding platform behavior to Python channel files.

Acceptance:

- SDD specs exist under `docs/sdd/specs/agent-architecture`.
- `services/message-gateway/architecture_test.go` enforces Go layer imports.
- New platform features cite the relevant SDD spec.

## Phase 1: Define Contracts

Goal:

- Stabilize `AgentInboundEvent`, `AgentDecision`, `MediaAsset`, and `AgentJob`
  schemas.
- Add golden JSON fixtures shared by Go and Python tests.
- Stabilize `MemoryExtractJob`, `RagIngestJob`, `GroupThread`, and source
  citation schemas before implementing group memory migration.

Acceptance:

- Go can validate inbound/outbound/job fixtures.
- Python can parse the same fixtures into typed dataclasses.
- Dashboard can render fixture messages and asset links without live platforms.
- Golden group-message replay cases exist for hardware DIY, game攻略, image/file
  evidence, conflicting claims, and unanswered questions.

## Phase 2: Go as Shadow Gateway

Goal:

- Keep current Python QQ/Telegram channels running.
- Mirror inbound/outbound events into Go for audit, ledger, and dashboard.

Acceptance:

- For each QQ private/group message, Go audit receives the same event id.
- Go loop guard can explain allow/observe/drop decisions in shadow mode.
- No user-visible routing change yet.

## Phase 3: Move QQ/Telegram Inbound Routing to Go

Goal:

- Go owns account registry, routing bindings, loop guard, and durable inbox.
- Python consumes only normalized allowed events.

Acceptance:

- Two QQ accounts can receive messages concurrently.
- Plain bot-to-bot messages are observed but not replied to.
- Explicit `/ask` or bot protocol messages reach Python.
- Per-account replies cannot route through the wrong QQ account.

## Phase 4: Move Outbox and Delivery to Go

Goal:

- Python returns `AgentDecision`; Go sends platform messages and files.
- Outbox handles retry, failure, and delivery receipts.

Acceptance:

- Image generation result is sent through the same account that received the
  request.
- Failed sends are visible in dashboard and audit.
- Python no longer calls `send_private_text`, `send_group_text`, or equivalent
  platform SDK methods directly for migrated channels.

Current implementation slice:

- Go owns `OutboxDelivery` state for `/v1/outbound`.
- `/v1/outbox` exposes queued/dispatching/succeeded/failed/dead-letter state.
- Actual QQ/Telegram platform SDK dispatch remains on the Python compatibility
  path until a reviewed adapter cutover exists.

## Phase 5: Move Media/File Registry to Go

Goal:

- Go downloads/registers QQ images/files and exposes local dashboard URLs.
- Python receives asset ids and optional summaries, not raw platform URLs.

Acceptance:

- Dashboard shows clickable images/files for QQ groups and private chats.
- Python vision tools can request asset bytes by id.
- Asset retention is configurable per channel/group.

## Phase 6: Refactor Python Agent Loop

Goal:

- Split large turn files into a state machine and context pipeline.
- Add stable core/deferred tool catalog.

Acceptance:

- `agent/core/passive_turn.py` no longer owns context compression, model call,
  tool execution, and commit logic in one file.
- Context pipeline stages have unit tests.
- Deferred tools can be discovered without changing the provider tool array.
- Tool-policy decisions are traceable.

## Phase 7: Jobize Image/RAG/Memory Work

Goal:

- Go owns job state and retry.
- Python workers execute image generation, RAG ingestion, and group-memory
  extraction.
- Group message processing follows `docs/sdd/specs/group-message-memory`.

Acceptance:

- Multiple image jobs from different users can run without blocking the main
  inbound queue.
- RAG ingestion jobs have pending/running/succeeded/failed states.
- Failed jobs can retry or dead-letter with visible error messages.
- Every extracted group memory item cites source messages/assets.
- Observe-only groups still emit no outbound replies during extraction/RAG.

## Phase 8: Agent Multiplexing

Goal:

- Introduce `AgentId` and routing bindings.
- Support specialized agents for hardware groups, game guide extraction, and
  personal chat.

Acceptance:

- Each agent has its own workspace, memory scope, skill allowlist, and sessions.
- One QQ group can be bound to an observe-only curator agent.
- A private owner chat can route to the main assistant agent.

## Deferred Work

- Replace in-memory Go store with SQLite/Postgres plus NATS/Redis Streams.
- Move scheduler infrastructure to Go while keeping proactive content selection
  in Python.
- Add OpenTelemetry traces once boundaries stabilize.
