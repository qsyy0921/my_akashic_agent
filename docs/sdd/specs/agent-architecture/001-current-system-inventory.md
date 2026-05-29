# SPEC-001: Current System Inventory

## Status

Draft

## Context

Akashic started as a Python agent service and has accumulated platform
channels, memory, proactive jobs, tool routing, dashboards, and experiments in
one process. This makes feature work fast, but it also creates hidden coupling:
platform bugs can block model reasoning, long image-generation turns can block
other users, and channel-specific safety rules live next to LLM runtime code.

This inventory is the baseline for deciding what should stay in Python and what
should move to Go.

## Current Module Map

| Area | Main Paths | Current Responsibility | Risk |
| --- | --- | --- | --- |
| Entry/bootstrap | `main.py`, `bootstrap/*` | start runtime, providers, channels, dashboard, memory, proactive loops | `AppRuntime` wires too many concerns |
| Message bus | `bus/*` | in-process inbound/outbound queues and lifecycle event bus | no durable queue, no per-conversation concurrency control |
| Channels | `infra/channels/*` | CLI, Telegram, QQ/NapCat, Feishu/WeChat webhooks, file/media handling | platform IO and safety mixed with agent session logic |
| Agent core | `agent/core/*`, `agent/looping/*`, `agent/lifecycle/*` | passive/proactive/drift loops, prompt rendering, hooks, response parsing | large files and several loop variants duplicate state logic |
| Tools | `agent/tools/*`, `plugins/*`, `skills/*` | tool schemas, tool discovery, shell/filesystem/web/image/chatgpt integrations | tool capability, permissions, and runtime IO are only partially separated |
| Memory | `core/memory/*`, `memory2/*`, `group_memory/*`, `plugins/default_memory/*` | markdown memory, semantic memory, group memory, retrieval and consolidation | multiple memory engines and stores overlap |
| Proactive | `proactive_v2/*` | feeds, scoring, proactive message decisions | same LLM/tool loop ideas as passive turn but separate implementation |
| Dashboard | `bootstrap/dashboard_api.py`, `static/dashboard/*`, `frontend/dashboard/*` | status, sessions, memory admin, attachment display | API is large and mixed with Python runtime internals |
| Go gateway | `services/agent-runtime/*` | first DDD skeleton for message normalization, loop guard, outbound ledger, image jobs | not yet wired as the system boundary |
| Evaluation | `eval/*`, `tests/*` | memory/RAG and runtime tests | useful, but not yet tied to architecture gates |

## Oversized Files

Files above roughly 500 lines should not receive new responsibilities without a
split-first review:

| File | Approx Lines | Action |
| --- | ---: | --- |
| `agent/core/passive_turn.py` | 1725 | split into context pipeline, model loop, tool loop, commit stage |
| `memory2/store.py` | 1687 | split persistence, schema migration, query APIs |
| `bootstrap/dashboard_api.py` | 1339 | split dashboard resources by bounded context |
| `plugins/default_memory/engine.py` | 1322 | split extraction, consolidation, retrieval |
| `infra/channels/qq_channel.py` | 1125 | move platform IO/routing/safety to Go gateway |
| `agent/core/proactive_turn.py` | 1061 | merge loop primitives with passive through shared turn state machine |
| `agent/tools/shell.py` | 1042 | split policy, execution, rendering, Windows/POSIX adapters |
| `infra/channels/telegram_channel.py` | 955 | move adapter delivery and media handling to Go gateway |

## Current Pain Points

- Python has a single inbound consumer path, so long tool runs can delay other
  users and other QQ accounts.
- Channel routing is stringly typed (`qq`, `qq_2365524513`, `gqq:*`) and not
  centrally validated.
- Private bot-to-bot loop protection exists in Python, but send ledgers,
  protocol tags, and delivery receipts should be durable infrastructure.
- Media and file attachments are downloaded and summarized in channel adapters,
  then partly exposed through dashboard APIs. Asset metadata should have a
  single owner.
- Tool discovery exists, but the stable-core/deferred-tool split is not strict
  enough to protect prompt cache and model attention.
- Memory systems are powerful but overlapping. Group memory, semantic memory,
  markdown memory, and RAG should share ingestion/job boundaries.

## Ownership Decision

Use this rule before adding code:

- If it needs low-latency IO, durable delivery, idempotency, routing, rate
  limits, retries, audit, account state, or platform protocol handling, it
  belongs in Go.
- If it needs model reasoning, prompt construction, tool semantics, memory/RAG
  ranking, extraction, skill instructions, or provider-specific LLM behavior, it
  belongs in Python.
- If it is presentation-only, it belongs in the frontend, backed by explicit
  Go/Python APIs instead of reading internal files directly.
