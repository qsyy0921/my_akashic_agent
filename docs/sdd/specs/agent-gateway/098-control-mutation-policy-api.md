# Control Mutation Policy API

## Context

Go now owns `ControlMutationPolicy` and the preflight service uses it to block
unsupported control-plane mutation target/action pairs. The policy should also
be visible through a stable runtime API so operators, dashboards and future
executors can inspect the supported mutation surface without duplicating the
allowlist in Python or frontend code.

## Ownership Boundary

Go owns:

- supported control mutation target/action policy;
- deterministic policy query API;
- side-effect-free runtime API response.

Python owns:

- AI model/tool/RAG/image execution;
- no policy authoring or mutation authorization.

Dashboard owns:

- optional read-only display of the policy response;
- no local policy copy.

## Goals

- Add `GET /v1/control-mutations/policy`.
- Return all supported target/action intents.
- Support optional `target_kind` query for one target.
- Return `allowed=false` and stable blocker for unsupported target.
- Keep `side_effect=none`.

## Non-Goals

- No mutation execution.
- No approval or audit creation.
- No config/env/cutover/worker/AgentJob/MQ/outbox/AI side effects.
- No dashboard UI redesign.

## Acceptance

- Domain policy exposes deterministic supported intent listing.
- App service test covers all-policy and filtered unsupported target.
- HTTP handler test covers `GET /v1/control-mutations/policy`.
- `go test ./domain/service ./app/service ./trigger/http` passes.
