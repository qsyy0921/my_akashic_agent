# Implementation Freeze Gate

## Status

Open for reviewed, non-production migration slices.

## Rule

Do not cut over production runtime behavior for the Go/Python architecture
split, QQ group memory, RAG, media registry, or multi-account routing until the
relevant SDD gates below are satisfied and the specific cutover has its own
review record.

Compatibility fixes may still be made when required to keep the current running
system working, but they must not add new architecture responsibilities to
already-overloaded Python channel or agent files.

## Gate Status

### Gate 1: Architecture Specs

- `docs/sdd/specs/agent-architecture` exists.
- `docs/sdd/specs/group-message-memory` exists.
- ADR records the Go/Python boundary.
- At least three review rounds are recorded.

Status: satisfied.

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

Status: satisfied for contract/fixture validation; add more fixtures as each
runtime boundary is migrated.

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

Status: satisfied for initial replay fixtures; expand before group-visible
features.

### Gate 4: Operational Safety

- Observe-only behavior is an invariant.
- Asset access policy is specified.
- LLM redaction stage is specified.
- Retry/dead-letter behavior is specified for long-running jobs.
- Account id is included in every route, job, asset, and citation.

Status: partially satisfied. Account id is required in message/outbox/job
contracts; media registry and scheduler cutovers still need dedicated checks.

## Explicitly Not Allowed Without A New Review

- Do not move QQ/Telegram adapters to Go.
- Do not replace the Python in-process message bus.
- Do not enable Go outbox as the production send path.
- Do not enable automatic group replies.
- Do not make RAG answers visible to a group chat.
- Do not add more responsibilities to `infra/channels/qq_channel.py`,
  `infra/channels/telegram_channel.py`, or `agent/core/passive_turn.py`.

## Allowed Migration Work

- Add non-production Go control-plane slices behind typed APIs.
- Add shadow-mode instrumentation.
- Add Go domain/app/infrastructure ports with tests.
- Add dashboard read-only views over Go-owned state.
- Add Python compatibility clients that mirror or query Go state.
- Review and refine architecture documents.

Production cutover still requires a new review note, rollback plan, tests, and
runtime verification.
