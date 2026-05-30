# Review: agent job lease token

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added `lease_token` to the Go `AgentJob` aggregate, query view, HTTP DTO, and
  JSON persistence.
- Go now generates a fresh token on each `Lease` / `LeaseNext` call and returns
  it to clients.
- `running`, `succeeded`, and `failed` transitions accept optional
  `lease_token`; when present, Go rejects stale or mismatched tokens before
  mutating state.
- Successful, failed, retry, and cancel transitions clear the active lease
  fields so terminal jobs do not keep an active token.
- Python image, knowledge, and RAG-eval workers now pass the lease token through
  result writeback. Legacy no-token calls remain compatible.

Tests run:
- `go test ./...`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_agent_gateway_image_worker.py tests\test_agent_gateway_knowledge_worker.py tests\test_agent_gateway_rag_eval_worker.py tests\test_agent_gateway_knowledge_worker_smoke.py -q --basetemp .tmp\pytest-agent-token`

Findings:
- The first Python test run using the default Windows temp directory failed
  with `PermissionError` under `AppData\Local\Temp\pytest-of-qsyy0921`; rerun
  with a repo-local `--basetemp` passed.
- Token fencing is deliberately compatibility-mode: a supplied bad token is
  rejected, but an omitted token is still accepted until all workers are known
  to be updated.

Decision:
Treat lease-token fencing as the first result-ack building block. Do not move
generic `agent_job` work to NATS external lease until strict token mode,
heartbeat/renewal, exact queue-id leasing, and duplicate-delivery contract smoke
exist.

Follow-ups:
- Add a strict token cutover flag once all Python workers are upgraded.
- Add worker heartbeat / lease renew before any long-running `agent_job` queue
  consumer can hold a NATS delivery open.
