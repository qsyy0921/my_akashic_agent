# Review: Control Mutation Policy API

Spec: `docs/sdd/specs/agent-gateway/098-control-mutation-policy-api.md`

## Implementation Summary

- Added deterministic domain listing for supported control mutation intents.
- Added `ControlMutationPolicyService`.
- Added input port `ControlMutationPolicyViewer`.
- Added `GET /v1/control-mutations/policy`.
- Added domain, app service and HTTP handler tests for all-policy and filtered
  unsupported target paths.

## Boundary Check

- Go owns the allowlist and exposes it through a stable read-only runtime API.
- Python and dashboard do not maintain a duplicate mutation policy.
- The API is read-only: no approval creation, no audit creation, no config/env
  mutation, no cutover, no worker startup, no AgentJob/MQ/outbox/AI side effect.

## Tests

- `go test ./domain/service ./app/service ./trigger/http`

## Findings

- The app service now uses a dedicated domain `SupportedIntent` query instead
  of inferring policy through a fake action.
- The API makes the future executor contract inspectable before any real
  mutation executor exists.

## Decision

Accept.

## Follow-Ups

- Future dashboard drilldown can render this API directly.
- Future executors must call the same policy/preflight path before applying any
  real control-plane mutation.
