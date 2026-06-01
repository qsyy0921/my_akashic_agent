# Review: Control Mutation Policy

Spec: `docs/sdd/specs/agent-gateway/097-control-mutation-policy.md`

## Implementation Summary

- Added Go domain service `ControlMutationPolicy`.
- Added explicit supported target/action allowlist for future control-plane
  mutations.
- Integrated the policy into `ControlMutationPreflightService`.
- Preflight now blocks unsupported target/action before returning `ready=true`.
- Added domain, app service and HTTP handler tests.

## Boundary Check

- Go owns mutation target/action authorization and preflight gating.
- Python still owns AI workers, provider/model execution, prompt/tool/RAG/image
  workflows and does not participate in control-plane authorization.
- The endpoint remains read-only and side-effect-free.

## Tests

- `go test ./domain/service ./app/service ./trigger/http`

## Findings

- The policy prevents a future executor from treating arbitrary
  `target_kind/action` strings as executable just because an approval exists.
- The allowlist is intentionally small; expanding it should require SDD review
  and tests.

## Decision

Accept.

## Follow-Ups

- Future mutation executors must bind to this allowlist, an active approval,
  mutation audit id, rate/fuse controls and rollback records before applying any
  real control-plane change.
