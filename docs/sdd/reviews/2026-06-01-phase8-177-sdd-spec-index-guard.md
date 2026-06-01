# Phase 8 Review 177: SDD Spec Index Guard

## Scope

- Add a repository governance test for `docs/sdd/specs/agent-gateway/000-index.md`.
- Update the index so all current agent-gateway specs are referenced by exact filename.
- Keep the slice documentation/test-only; no runtime API, MQ, worker, media, RAG, or AI behavior changes.

## Design Check

- The guard is intentionally a Python repository test because it validates SDD/document contracts, not Go runtime behavior.
- The runtime boundary remains unchanged: Go continues to own deterministic infrastructure under `services/agent-runtime`; Python continues to own AI workers and fast-moving model/tool pipelines.
- The test checks exact filenames, which catches both newly added specs and renamed specs that have not been reflected in the index.

## Verification

- `uv run pytest tests/test_sdd_spec_index.py -q`
- `uv run pytest tests/test_sdd_contract_fixtures.py -q`
- `go test ./...` from `services/agent-runtime`
- `git diff --check`

## Risk

- The test enforces discoverability, not the quality of each spec summary. Review discipline is still needed for meaningful descriptions.

## Result

Accepted. Agent-gateway SDD specs now have an automated index coverage check.
