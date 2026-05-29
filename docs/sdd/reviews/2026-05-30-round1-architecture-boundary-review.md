# SDD Review Round 1: Architecture Boundary

## Scope

Review whether the proposed architecture separates Go infrastructure from
Python agent intelligence without copying OpenClaw, Hermes, or Claude Code
blindly.

Reviewed:

- `docs/sdd/specs/agent-architecture/*`
- `docs/sdd/specs/group-message-memory/*`
- `docs/sdd/adr/0002-python-go-agent-boundary.md`

## External Review Attempt

Attempted to use the Claude review tool for an independent review, but the tool
returned `Not logged in - Please run /login`. This review is therefore an
internal SDD review, not an external reviewer approval.

## Findings

### P0

None.

### P1

1. **Implementation must remain frozen until lower-level contracts exist.**
   The high-level Go/Python boundary is sound, but implementation should not
   continue from this alone. Phase 1 contract fixtures are required first:
   `AgentInboundEvent`, `AgentDecision`, `MediaAsset`, `MemoryExtractJob`,
   `RagIngestJob`, `GroupThread`, and citations.

2. **Existing Python channel files are already overloaded.**
   `infra/channels/qq_channel.py` and `telegram_channel.py` remain too large and
   should receive only compatibility fixes until Go shadow gateway is ready.

3. **Go is not a replacement for agent intelligence.**
   The docs correctly keep LLM calls, prompt/context pipeline, tools, memory,
   and RAG in Python. This should stay explicit in every migration spec.

### P2

1. The dashboard ownership split is reasonable but needs API specs before
   implementation.
2. Agent multiplexing is correctly deferred until routing and contract fixtures
   exist.
3. Scheduler infrastructure can move to Go later; proactive content reasoning
   should remain Python-side.

## Decision

**Pass for high-level architecture.**

This is not authorization to implement the migration. It authorizes writing the
next lower-level contract specs and fixtures.

## Required Follow-Up Before Code Migration

- Write contract fixture spec.
- Define Go/Python compatibility tests for shared JSON.
- Define shadow-mode telemetry for QQ groups.
- Add an implementation freeze checklist to SDD.

