# SPEC-003: Python / Go Boundary

## Status

Draft

## Context

The project now needs to support multiple QQ accounts, Telegram, group
observation, media/file browsing, image-generation jobs, group memory, and RAG.
These have different runtime needs. Combining all of them in one Python process
creates backpressure and makes safety rules hard to test.

## Go Responsibilities

Go is the backend infrastructure boundary. It should own behavior that must be
durable, concurrent, protocol-aware, and easy to test without an LLM.

### Bounded Contexts

| Context | Domain Concepts | Why Go |
| --- | --- | --- |
| Channel accounts | `AccountId`, `Platform`, `LoginState`, `CredentialRef` | multiple QQ/Telegram accounts need deterministic ownership |
| Routing | `AgentId`, `Binding`, `Conversation`, `ConversationType` | prevents stringly typed `qq_*` routing bugs |
| Inbound normalization | `MessageEnvelope`, `Sender`, `Attachment`, `Provenance` | every platform emits one schema |
| Loop protection | `BotProtocol`, `Nonce`, `Hop`, `SendRecord` | pure domain logic, easy to test |
| Durable queues | `InboxEvent`, `OutboxEvent`, `RetryPolicy`, `DeadLetter` | removes Python single-queue bottleneck |
| Outbound dispatch | `OutboundMessage`, `DeliveryReceipt`, `OutboxLedger` | exactly-once-ish delivery and retries |
| Media/file registry | `MediaAsset`, `FileAsset`, `AssetURL`, `RetentionPolicy` | dashboard and agent need stable asset links |
| Job orchestration | `ImageJob`, `RagIngestJob`, `MemoryExtractJob` | long-running jobs need status, retry, cancellation |
| Observation | `ObservePolicy`, `GroupWatch`, `SamplingWindow` | QQ groups can be observed without triggering replies |
| Audit | `AuditEvent`, `PolicyDecision`, `TraceRef` | replayable state for debugging and SDD review |
| Scheduler infrastructure | `Schedule`, `Tick`, `Lease`, `DeliveryTarget` | reliable timing and multi-worker coordination |

### Go App Services

- `MessageIngestService`: validate and normalize inbound messages.
- `RoutingService`: resolve `(platform, accountId, peer)` to `agentId`.
- `LoopGuardService`: decide allow/observe/drop before Python sees the event.
- `OutboxService`: store outbound request, apply route, dispatch via adapter.
- `MediaService`: download/register media and return stable local URLs.
- `JobService`: create/update/query long-running jobs.
- `AuditService`: append events and expose dashboard queries.
- `SchedulerService`: emit durable ticks; Python decides content when an LLM is
  required.

## Python Responsibilities

Python is the agent intelligence runtime. It should own behavior where model
quality, prompt engineering, tool semantics, and ML/RAG libraries matter.

| Area | Owner Paths | Reason |
| --- | --- | --- |
| Agent turn state machine | `agent/core`, `agent/lifecycle`, `agent/turns` | LLM/tool loop logic and provider-specific recovery |
| Prompt/context pipeline | `agent/context.py`, future `agent/context_pipeline/*` | compression, memory injection, tool hints |
| Provider/model routing | `agent/provider.py`, `infra/providers/*` | MiMo, vision model, embedding, fallback, streaming |
| Tool runtime | `agent/tools/*`, `agent/tool_hooks/*` | Python ecosystem and rich side-effect tools |
| Memory extraction | `core/memory`, `memory2`, `group_memory` | embeddings, LLM extraction, RAG, summarization |
| Skills/procedures | `skills/*`, `agent/skills.py` | natural-language operational guidance |
| RAG evaluation | `eval/*` | Python ML/eval ecosystem |
| Image generation execution | `agent/tools/chatgpt_proxy.py` initially | browser/profile and provider-specific implementation |
| Proactive reasoning | `proactive_v2/*` initially | LLM scoring and natural-language delivery decisions |
| Multimodal understanding | vision/OCR adapters and media summarizers | provider-specific OCR/VLM behavior and prompt tuning |
| Embedding and rerank experiments | memory/RAG adapters and eval scripts | model choice, chunking, recall/precision tradeoffs change frequently |
| Compatibility mirrors | runtime bridge clients | keep old dashboard/runtime paths alive during migration, not source of truth |

Python should consume Go events through a narrow client:

```text
Go AgentGateway -> AgentInboundEvent -> Python AgentRuntime
Python AgentRuntime -> AgentDecision -> Go Outbox/Job APIs
```

## Frontend Responsibilities

Frontend should not infer business state from Python internals.

| Area | Source API |
| --- | --- |
| Message list | Go `ConversationQuery` |
| Asset links | Go `MediaAssetQuery` |
| Job status | Go `JobQuery` |
| Agent traces | Python `TraceQuery` mirrored to Go audit |
| Memory inspector | Python memory admin API, later mirrored summary in Go |

## Boundary Contracts

### AgentInboundEvent

```json
{
  "event_id": "qq:2365524513:private:1049511700:123",
  "agent_id": "main",
  "channel": {
    "platform": "qq",
    "account_id": "2365524513",
    "conversation_id": "1049511700",
    "conversation_type": "private"
  },
  "sender": {
    "id": "1049511700",
    "kind": "human|bot|self",
    "display_name": ""
  },
  "content": "/ask generate image",
  "attachments": [],
  "provenance": {
    "bot_protocol": null,
    "policy_decision": "allow"
  },
  "timestamp": "2026-05-30T00:40:00+08:00",
  "metadata": {}
}
```

### AgentDecision

```json
{
  "event_id": "agent:turn:uuid",
  "source_event_id": "qq:2365524513:private:1049511700:123",
  "agent_id": "main",
  "decision": "reply|observe|skip|create_job",
  "outbound": {
    "channel": {
      "platform": "qq",
      "account_id": "2365524513",
      "conversation_id": "1049511700",
      "conversation_type": "private"
    },
    "content": "queued image job",
    "attachments": []
  },
  "jobs": [],
  "trace_ref": "trace:uuid"
}
```

## Invariants

- Python never sends directly to a platform once Go adapter is enabled. It
  requests Go outbox dispatch.
- Go never calls an LLM for core routing decisions.
- Python owns AI-quality decisions, but not durable infrastructure truth once
  the matching Go domain API exists.
- Python compatibility mirrors are temporary fallback/read-through layers, not
  competing stores for leases, routing, queues, assets, or audit state.
- Every inbound and outbound event has an idempotency key.
- Every media/file attachment has a stable asset id before it is shown in the
  dashboard or passed to Python.
- Bot-to-bot messages require explicit protocol or configured trigger prefixes.
- Per-account routing cannot fall back from `qq_2365524513` to `qq` silently.
