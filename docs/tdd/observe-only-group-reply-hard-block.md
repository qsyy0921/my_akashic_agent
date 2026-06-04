# TDD: Observe-Only Group Reply Hard Block

## Coverage Targets

1. Delivery dispatch rejects exact observe-only QQ group routes with
   `reply_allowed=false`.
2. Non-matching or private routes are still plannable.
3. Live runtime reproduces the block through
   `/v1/delivery-dispatch/readiness`.

## Tests

- `go test ./app/service ./cmd/agent-runtime`
- Live `GET /v1/observe-targets`
- Live `POST /v1/delivery-dispatch/readiness` for:
  - one exact observe-only QQ group route
  - one QQ private route
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
