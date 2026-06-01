# Phase 8 Review 178: Go Runtime Package Guard

## Scope

- Strengthen `services/agent-runtime/architecture_test.go`.
- Add a known top-level source-root guard for Go runtime packages.
- Keep the slice architecture-test-only; no runtime, API, MQ, worker, RAG, OCR/VLM, image generation, or Python behavior changes.

## Design Check

- The guard complements the existing import-direction test: dependency rules catch illegal imports, while the new source-root guard catches accidental top-level packages before they become architecture debt.
- Allowed roots match the current pragmatic DDD + hexagonal structure: `api`, `app`, `cmd`, `domain`, `infrastructure`, `smoke`, `trigger`, and `types`.
- `cmd` remains the composition root and is not constrained like inner layers.

## Verification

- `go test .` from `services/agent-runtime`
- `go test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_spec_index.py -q`
- `git diff --check`

## Risk

- Future legitimate top-level roots will require an explicit SDD update and test allowlist update. That is intentional because top-level package creation changes the architecture boundary.

## Result

Accepted. Go runtime source layout now has an automated guard against accidental top-level package drift.
