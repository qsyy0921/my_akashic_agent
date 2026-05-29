# Implementation Freeze Gate

## Status

Active for architecture migration work.

## Rule

Do not implement or migrate runtime behavior for the Go/Python architecture
split, QQ group memory, RAG, media registry, or multi-account routing until the
relevant SDD gates below are satisfied.

Compatibility fixes may still be made when required to keep the current running
system working, but they must not add new architecture responsibilities to
already-overloaded Python channel or agent files.

## Required Gates Before Implementation

### Gate 1: Architecture Specs

- `docs/sdd/specs/agent-architecture` exists.
- `docs/sdd/specs/group-message-memory` exists.
- ADR records the Go/Python boundary.
- At least three review rounds are recorded.

### Gate 2: Contract Fixtures

- Shared JSON fixtures are specified for:
  - `MessageEnvelope`
  - `AgentInboundEvent`
  - `AgentDecision`
  - `MediaAsset`
  - `AgentJob`
  - `MemoryExtractJob`
  - `RagIngestJob`
  - `GroupThread`
  - source citation records
- Go and Python compatibility tests are planned before code migration.

### Gate 3: Group Replay Cases

- Golden replay cases are specified for:
  - hardware DIY recommendations;
  - game攻略;
  - image-only evidence;
  - file attachment evidence;
  - conflicting claims;
  - outdated version-sensitive claims;
  - unresolved questions;
  - observe-only no-reply behavior.

### Gate 4: Operational Safety

- Observe-only behavior is an invariant.
- Asset access policy is specified.
- LLM redaction stage is specified.
- Retry/dead-letter behavior is specified for long-running jobs.
- Account id is included in every route, job, asset, and citation.

## Explicitly Not Allowed Yet

- Do not move QQ/Telegram adapters to Go.
- Do not replace the Python in-process message bus.
- Do not enable Go outbox as the production send path.
- Do not enable automatic group replies.
- Do not make RAG answers visible to a group chat.
- Do not add more responsibilities to `infra/channels/qq_channel.py`,
  `infra/channels/telegram_channel.py`, or `agent/core/passive_turn.py`.

## Allowed Next Work

- Write contract fixture specs.
- Write golden replay specs.
- Add tests that validate existing docs/contracts without migrating behavior.
- Add shadow-mode instrumentation specs.
- Review and refine architecture documents.

