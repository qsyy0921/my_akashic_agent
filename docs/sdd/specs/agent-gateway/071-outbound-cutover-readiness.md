# 071 Outbound Cutover Readiness

## Context

Go already owns the outbox state machine, OneBot/Telegram delivery adapters,
adapter diagnostics, manual health probes, and delivery smoke readiness. The
next operational question is broader than adapter support:

Can outbound platform sending be safely handed to Go for QQ/NapCat channels?

The answer requires multiple existing signals:

- sanitized runtime config: expected OneBot channels and missing aliases;
- delivery smoke readiness: private/group text/media dispatch planning and
  adapter support;
- queue backend execution owner: local state-store worker, NATS external lease,
  or state-store API only;
- runtime worker diagnostics: whether the actual Go outbox execution path is
  running.

## Decision

Add a read-only endpoint:

```text
POST /v1/outbound-cutover/readiness
```

The endpoint accepts the same request shape as delivery smoke readiness, so an
operator can override cases, group ids, channel mapping, or synthetic media
coverage. Empty body uses the env-derived default smoke matrix.

The readiness result returns:

- `ready`, `reason`, `blockers`;
- OneBot readiness and expected/missing channels;
- delivery smoke readiness summary and detail;
- outbox execution owner;
- local Go outbox worker readiness;
- NATS external lease outbox readiness;
- `execution_ready`, which is true when either local Go outbox worker or NATS
  external lease can execute `outbox_delivery`;
- `side_effect=none`.

Readiness is blocked when OneBot config is incomplete, smoke planning/adapters
are not ready, or no Go execution path can actually deliver queued outbox
records.

## Boundary

### Go owns

- deterministic cutover preflight;
- adapter routing and smoke planning checks;
- outbox execution owner and worker state aggregation;
- HTTP API for dashboard/operator consumption.

### Python owns

- AI reasoning, prompt/tool execution and message creation;
- compatibility callers that enqueue or directly send until cutover is
  explicitly enabled;
- any provider-specific platform workaround outside Go adapter contracts.

## Non-goals

- No platform messages are sent.
- No env vars or worker flags are mutated.
- No automatic cutover to Go local worker or NATS external lease.
- No Python behavior changes in this slice.

## Validation

- Service tests cover ready and blocked states.
- HTTP tests verify POST-only behavior and no adapter dispatch side effects.
- Main wiring tests must compile.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check` must
  pass.
