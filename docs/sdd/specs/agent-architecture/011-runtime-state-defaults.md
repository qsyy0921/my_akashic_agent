# Runtime State Defaults

## Status

Accepted

## Problem

The Go `agent-runtime` owns deterministic infrastructure state: observe target
configuration, receiver status diagnostics, inbox events, media assets, send
ledger entries, outbox deliveries, generic jobs, job events, knowledge
checkpoints, and proactive scheduling state. Most bounded contexts already had a
file-backed adapter, but the process default was still memory-backed unless
every store was configured separately.

That default made local observe-only QQ validation non-deterministic:

- restarting `agent-runtime` cleared sampled inbox/media state;
- `/v1/observe-capture-diagnostics` could not prove image/file coverage after a
  restart;
- generic job and RAG checkpoint recovery depended on operators remembering all
  per-store environment variables.

## Decision

`agent-runtime` now uses a common file-backed default directory for unconfigured
state stores:

```text
.akashic-workspace/agent-runtime
```

The process discovers the Akashic repo root from the current working directory
or executable path. If discovery fails, it falls back to the current working
directory. The default files are:

```text
agent-jobs.json
agent-job-events.jsonl
observe-targets.json
receiver-statuses.json
receiver-leases.json
inbound-dedupe.json
media-assets.json
send-ledger.json
outbox.json
outbox-events.jsonl
inbox.json
knowledge-checkpoints.json
proactive-state.json
```

Operators can override the common directory with:

```text
AKASHIC_RUNTIME_STATE_DIR=/path/to/state-dir
```

Operators can disable the common default with:

```text
AKASHIC_RUNTIME_STATE_DIR=memory
```

Per-store overrides keep priority:

```text
AKASHIC_INBOX_DSN
AKASHIC_INBOX_PATH
AKASHIC_MEDIA_ASSETS_DSN
AKASHIC_MEDIA_ASSETS_PATH
AKASHIC_OBSERVE_TARGETS_DSN
AKASHIC_OBSERVE_TARGETS_PATH
AKASHIC_RECEIVER_STATUSES_DSN
AKASHIC_RECEIVER_STATUSES_PATH
AKASHIC_RECEIVER_LEASES_DSN
AKASHIC_RECEIVER_LEASES_PATH
AKASHIC_INBOUND_DEDUPE_DSN
AKASHIC_INBOUND_DEDUPE_PATH
...
```

The existing `*_DSN=memory` convention still disables persistence for one store
without affecting the others.

## Boundaries

This is still a local file-backed development/default store, not the external MQ
or distributed database cutover. NATS JetStream remains a work-notification and
external-lease candidate; Go state stores remain authoritative until a separate
cutover gate is satisfied.

Python continues to own AI behavior: model calls, OCR/VLM, RAGFlow upload,
prompting, and memory extraction heuristics. It should rely on Go state APIs for
durable cursors and raw message replay instead of keeping parallel control-plane
state.

## Acceptance

- A fresh `agent-runtime` run without per-store env creates file-backed state
  under `.akashic-workspace/agent-runtime`.
- Observe targets synced from Python survive `agent-runtime` restarts without
  requiring an immediate Python resync.
- `AKASHIC_RUNTIME_STATE_DIR` redirects all unconfigured stores.
- `AKASHIC_RUNTIME_STATE_DIR=memory` keeps unconfigured stores ephemeral.
- `AKASHIC_*_DSN=memory` still wins for a specific store.
- Inbox events survive repository re-open through the default runtime state
  directory.
- `/v1/runtime-config` exposes `AKASHIC_RUNTIME_STATE_DIR` presence in the
  sanitized environment snapshot.

## Risks

- Local state files can grow over time. The directory is already ignored by Git;
  retention and compaction should be handled in a later operational slice.
- Existing operators who depended on memory-only defaults should set
  `AKASHIC_RUNTIME_STATE_DIR=memory` explicitly.
