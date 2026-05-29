# ADR-0002: Split Agent Intelligence From Messaging Infrastructure

## Status

Accepted

## Context

Akashic now needs to handle multi-account QQ, Telegram, group observation,
image-generation jobs, group memory, RAG, and dashboard media browsing. The
original Python-first runtime is productive but has accumulated infrastructure
responsibilities in channel adapters and bootstrap code.

The project also needs a story that can stand up technically: Go should not be
added just for resume optics, and Python should not keep absorbing IO-heavy
backend responsibilities.

## Decision

Use a two-runtime architecture:

- Go owns durable backend infrastructure using DDD plus hexagonal architecture.
- Python owns agent intelligence, model reasoning, tools, memory, RAG, and
  skills.

Go packages keep the existing layout:

```text
api
app
domain
infrastructure
trigger
types
```

Python is refactored toward an agent runtime layout:

```text
intake
context_pipeline
turn_state
reasoner
tool_runtime
memory_runtime
skills
decision
workers
```

The boundary contract is event-based:

```text
Go AgentGateway -> AgentInboundEvent -> Python AgentRuntime
Python AgentRuntime -> AgentDecision / JobResult -> Go Outbox / JobService
```

## Consequences

Positive:

- Multiple QQ/Telegram accounts can be routed and audited consistently.
- Long-running image/RAG jobs stop blocking the main agent inbox.
- Bot-to-bot loop protection becomes durable and testable without an LLM.
- Dashboard can show messages, files, jobs, and audit from typed APIs.
- Python remains free to use the best model/RAG/tooling ecosystem.

Negative:

- Requires schema discipline and cross-runtime tests.
- Local development needs both Python runtime and Go gateway.
- Some existing direct channel sends must be migrated gradually.

## Non-Goals

- Do not rewrite the whole system at once.
- Do not move LLM prompt engineering, memory extraction, or RAG ranking to Go.
- Do not replace all Python platform adapters until shadow-mode audit proves the
  Go path.

## Review Rules

- New platform/channel/account logic goes to Go or needs an explicit exception.
- New model/tool/memory logic stays in Python or needs an explicit exception.
- Any file already above 500 lines requires split-first review before receiving
  a new responsibility.
- Every migrated boundary must have golden JSON fixtures shared by Go and
  Python tests.
- Group-memory and RAG implementation cannot start until
  `docs/sdd/specs/group-message-memory` has passed architecture, data/RAG, and
  operational safety reviews.
