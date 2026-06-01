# SPEC-121: Go Runtime Package Guard

## Status

Accepted for the current iteration.

## Context

`services/agent-runtime` already has a dependency-direction test for the main
DDD layers, but it does not prevent future Go files from being added under
arbitrary new top-level directories. As the runtime grows, accidental top-level
packages can weaken the intended DDD + hexagonal shape and make ownership harder
to reason about.

## Boundary Analysis

Go owns:

- runtime package layout and compile-time architecture guardrails under
  `services/agent-runtime`;
- DDD/hexagonal boundary enforcement for deterministic infrastructure code.

Python owns:

- AI runtime, model/tool/prompt code, and repository-level document governance
  tests outside the Go module.

Out of scope:

- moving existing runtime behavior;
- changing HTTP/MQ/API contracts;
- changing Python AI workers, RAG, OCR/VLM, image generation, or provider
  routing.

## Decision

Extend `services/agent-runtime/architecture_test.go` with a package root guard:
non-test Go source files must live under one of the intentional top-level roots:

- `api`
- `app`
- `cmd`
- `domain`
- `infrastructure`
- `smoke`
- `trigger`
- `types`

This complements the existing import-direction test. It does not restrict
composition-root wiring in `cmd`, and it does not change runtime behavior.

## Acceptance

- `go test .` in `services/agent-runtime` catches unknown top-level Go source
  roots.
- `go test ./...` still passes.
- SDD index, DONE, LIVE_CHECKS, and review docs are updated.
- TODO is cleared at the end of the iteration.
