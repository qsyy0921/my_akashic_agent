# Review: runtime contract fixtures

Spec: `docs/sdd/specs/agent-architecture/006-contract-fixtures.md`

Implementation summary:

- Added shared Go/Python JSON fixtures for knowledge checkpoints, inbox replay, outbox delivery, media asset content, and agent job event streams.
- Extended Go fixture tests to assert current Go-owned runtime boundary fields are present and round-trip through `dto.ContractFixture`.
- Extended Python fixture tests to keep the manifest exact and assert behavior-critical fields for replay, delivery, content access, and job event ordering.

Tests run:

- `go test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_sdd_contract_fixtures.py -q --basetemp .tmp/pytest-runtime-contract-fixtures`

Findings:

- The generic `ContractFixture` model correctly preserves unknown fields, so new runtime boundary payloads can evolve without forcing every field into shared DTOs immediately.
- `MediaAssetContent` remains routed to an agent in the shared validator, so the fixture includes `agent_id` even though content serving is implemented by Go infrastructure.

Decision:

- Accepted as a low-risk contract-only migration slice. It adds no runtime side effects and does not change QQ observation or delivery behavior.

Follow-ups:

- Add contract fixtures for live adapter smoke results after explicit user approval for real QQ sends.
- Extend fixtures again when external queue backends or RAG evaluation jobs are introduced.
