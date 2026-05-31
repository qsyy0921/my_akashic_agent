# SPEC-010: Python Agent Runtime DDD Suitability

## Status

Accepted design guidance.

## Question

Should the Python side use the same DDD package structure as the Go
`agent-runtime`?

## Decision

Use lightweight ports-and-adapters plus pipeline/use-case boundaries in Python.
Do not force a full DDD package structure across the Python agent runtime.

The Go side should continue using tactical DDD with hexagonal boundaries because
it owns stable backend domains: message envelopes, inbox/outbox delivery,
send-ledger idempotency, loop guard decisions, media assets, generic jobs, and
future durable queues.

The Python side should keep these boundaries:

```text
agent runtime / orchestration
├── use-case services: turn handling, knowledge jobs, image jobs
├── ports: model provider, runtime client, memory store, tool registry
├── adapters: MiMo/OpenAI/ChatGPT proxy/RAGFlow/QQ/Telegram integration clients
├── pipelines: prompt rendering, tool calling, memory extraction, RAG
└── plugins/tools: rapidly changing AI capabilities
```

## Why Go Fits Tactical DDD

- Go code is now responsible for durable state machines and invariants:
  `InboxEvent`, `OutboxDelivery`, `AgentJob`, `MediaAsset`, `SendRecord`, and
  loop-guard decisions.
- Those objects need consistent transitions, idempotency, retries, and stable
  API contracts.
- DDD + hexagonal architecture keeps framework and storage details out of the
  domain model while still allowing file-backed, memory-backed, or later
  database-backed repositories.
- Go's explicit interfaces and package visibility work well for this style when
  kept lightweight.

## Why Python Should Not Be Full DDD

- The Python core is an AI/runtime system, not a stable CRUD/business-domain
  system. Most complexity is in model calls, prompt evolution, tool selection,
  multimodal processing, and RAG experiments.
- Full DDD would add ceremony around behavior that changes frequently. The
  aggregate boundaries would be unstable because prompt/tool pipelines often
  cross-cut memory, retrieval, model providers, and channel state.
- OpenClaw, Hermes, Claude Code-like agent runtimes generally use agent loop,
  tool registry, model adapter, session/memory, and event/pipeline modules, not
  Java-style DDD layers. Their architecture is closer to plugin runtime plus
  ports/adapters.
- Python tests and implementation velocity benefit from small protocols,
  dataclasses, service objects, and explicit pipeline stages more than from
  forcing every concept into entity/factory/domain-service packages.

## Python Rules Going Forward

- Keep model calls, prompt strategy, tool execution, extraction, RAG ranking,
  and fast AI experiments in Python.
- Keep provider-specific behavior in Python: MiMo/OpenAI/ChatGPT proxy quirks,
  streaming recovery, OCR/VLM prompts, image generation execution, embedding
  model selection, rerank thresholds, and eval harnesses.
- Define ports at integration boundaries, for example runtime job client,
  inbox message source, RAG indexer, image generation tool, and model provider.
- Use clear service classes for use cases such as `AgentRuntimeKnowledgeWorker`
  and `GroupMemoryService`.
- Avoid global utility sprawl by grouping code by capability and keeping adapters
  narrow.
- Do not let Python own durable infrastructure once Go has a matching domain and
  API.
- Treat Python local files/databases used during migration as compatibility
  mirrors or algorithm caches unless the spec explicitly declares them
  authoritative.

## Boundary Example

Group memory extraction should be split as:

```text
Go agent-runtime
  owns InboxEvent persistence, cursor-safe query, job lifecycle

Python group_memory
  owns strategy extraction, evolution, RAG retrieval, LLM/VLM enrichment

Python integration adapter
  reads /v1/inbox and maps runtime events to observed-message rows
```

This keeps Go's DDD model useful without making Python imitate a backend service
architecture that does not fit an agent runtime.
