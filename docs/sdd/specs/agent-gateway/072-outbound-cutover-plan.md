# 072 Outbound Cutover Plan

## Context

`/v1/outbound-cutover/readiness` can now answer whether QQ/NapCat outbound
delivery is ready to be handed to Go, but operators still need a structured
plan before changing runtime flags. Keeping that plan as notes in docs is too
easy to drift from the actual runtime configuration.

The plan should be a deterministic Go control-plane view because it depends on
runtime config, queue mode, worker state, adapter support and cutover gates. It
must not execute AI work, send platform messages or mutate environment
variables.

## Decision

Add a read-only endpoint:

```text
POST /v1/outbound-cutover/plan
```

The request accepts the same smoke matrix fields as readiness, plus an optional
`desired_execution_owner`:

- `auto`
- `go_local_outbox_worker`
- `nats_external_lease`

The response returns:

- current and desired execution owner;
- recommended execution owner;
- the embedded outbound cutover readiness result;
- required pre-cutover checks;
- enable/configuration steps;
- verification endpoints and smoke steps;
- rollback steps;
- blockers and `side_effect=none`.

The endpoint is allowed to recommend env keys and operational steps, but it must
not write env vars, enqueue outbox records, mutate jobs, or dispatch delivery
adapter calls.

## Boundary

### Go owns

- cutover readiness aggregation;
- deterministic cutover sequencing for local outbox worker and NATS external
  lease modes;
- operator-facing plan, verification and rollback instructions;
- JSON API consumed by dashboard or scripts.

### Python owns

- LLM/VLM/prompt/tool execution that creates outbound content;
- compatibility send paths before an operator explicitly enables Go delivery;
- any provider-specific AI fallback and prompt-level policy.

## Non-goals

- No platform sends.
- No environment mutation.
- No automatic worker start/stop.
- No switch from Python compatibility sending to Go sending.
- No new MQ adapter.

## Validation

- Service tests cover local worker recommendation and external lease blockers.
- HTTP test verifies request parsing, POST-only behavior and read-only result.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
