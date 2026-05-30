# SPEC-023: Observe Target Diagnostics

## Status

Implemented as a read-only Go runtime diagnostic slice.

## Context

QQ observation is currently configured in Python `config.toml` under
`channels.qq.groups` and account-specific `channels.qq.accounts[*].groups`.
The message collection path should keep running unchanged, but operators need a
Go-owned view of which groups are intentionally observe-only before moving more
runtime state into Go.

Without this view, group memory jobs, inbox metrics, and delivery smoke checks
can disagree about the intended observation surface.

## Decision

Python remains the TOML/config adapter. On startup, when `agent_runtime` is
enabled, Python builds a normalized observe-target snapshot from observe-only
QQ groups and syncs it to Go:

```text
PUT /v1/observe-targets/sync
```

Go owns the runtime state and read model:

```text
GET /v1/observe-targets
GET /v1/runtime-overview
```

The sync is source-bound. A `python_config` sync replaces only previous
`python_config` targets and preserves targets from other future sources. This
keeps room for later admin UI or dynamic discovery without turning Python back
into the owner of runtime diagnostics.

## Contract

Each target contains:

- `target_id`: stable id such as `qq:1049511700:group:27234224`;
- `channel`: kind, account id, conversation id, conversation type;
- `observe_only`: true for configured QQ observation groups;
- `reply_allowed`: forced to false when `observe_only=true`;
- `require_at`, `allow_from`, `enabled`, `source`, `metadata`;
- `side_effect=none` on the aggregate response.

Go aggregates totals for target count, enabled/disabled, observe-only,
reply-allowed, groups/private, and platform kind.

## Boundaries

Go owns:

- observe target validation and normalization;
- source-bound runtime storage;
- runtime overview summary/card semantics.

Python owns:

- reading TOML/dataclass config;
- translating configured observe-only QQ groups into the sync request;
- dashboard display normalization.

This slice must not change QQ/NapCat receiving, reply routing, memory
extraction, RAG ingestion, or platform sends.

## Acceptance

- Go exposes `PUT /v1/observe-targets/sync` and `GET /v1/observe-targets`.
- Python startup best-effort syncs observe-only QQ groups to Go when
  `agent_runtime.enabled=true`.
- Runtime overview includes an `Observe Targets` card and summary fields.
- Unit tests cover Go validation, source-bound sync, HTTP handler contract,
  Python client sync/list, Python target construction, and dashboard
  normalization.
