# Queue Topology Read Model

## Context

`/v1/queue-backend` already exposes detailed provider state, execution owner
fields and external lease gates. The payload is useful for diagnostics, but it
is not a concise topology view. Operators still need to mentally map:

- state-store vs external queue authority;
- outbox delivery execution owner;
- AgentJob execution / result-ack owner;
- provider capability readiness;
- external lease allowed and blocked work kinds.

This slice adds a Go-owned read-only topology endpoint. It is an operator/API
projection built from existing Go control-plane state.

## Boundary

Go owns this view because queue topology is deterministic infrastructure:

- provider and migration mode;
- selected/recommended capability;
- work-kind ownership for outbox and AgentJob;
- whether state-store, NATS external lease, Go local worker or Python worker is
  the current control/execution owner;
- blockers and cutover notes.

Python remains responsible for AI execution:

- AgentJob model/RAG/OCR/VLM/image workers;
- prompt/tool orchestration;
- provider-specific retry/fallback;
- result-ack client implementation when external lease is enabled.

The topology endpoint must not:

- publish/lease/ack/nack/term queue work;
- mutate env/config;
- start/stop workers;
- create AgentJob or outbox records;
- send QQ/Telegram messages;
- call model providers.

## Design

Add:

```text
GET /v1/queue-topology
```

The application service uses `QueueBackendViewer.Get(ctx)` and derives:

- `provider`, `mode`, `migration_phase`;
- `selected_provider`, `recommended_provider`;
- `nodes`: state store, external queue, Go outbox worker, Python AI worker,
  and provider-specific runtime nodes;
- `edges`: work-kind flow for `outbox_delivery` and `agent_job`;
- `work_kinds`: compact per-work-kind owner, queue source, execution owner,
  ack owner, gate state and blockers;
- `blockers`, `notes`, `side_effect=none`.

Rules:

- `outbox_delivery` execution owner comes from
  `QueueBackendView.OutboxExecutionOwner`;
- `agent_job` execution owner comes from
  `QueueBackendView.AgentJobExecutionOwner`;
- if external lease gate is present, allowed/blocked work kinds refine the
  gate state for each work kind;
- selected provider capability remains advisory and read-only.

## Verification

- `go test ./app/service -run TestQueueTopology -count=1 -v`
- `go test ./trigger/http -run TestQueueTopologyEndpointReturnsReadOnlyView -count=1 -v`
- `go test ./...`

## Risks

- This endpoint does not prove live NATS connectivity. It reflects current Go
  queue control-plane configuration and diagnostics.
- It is intentionally read-only; real cutover still requires the existing
  readiness/plan endpoints, live smoke and explicit operator env changes.
