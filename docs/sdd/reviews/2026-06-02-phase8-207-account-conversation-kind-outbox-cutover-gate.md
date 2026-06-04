# Phase 8.207 Review: account-conversation-kind outbox cutover gate

## What changed

- Added a route-scoped Go outbox gate:
  - `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE`
- Extended lease filtering, runtime config, queue-backend diagnostics, runtime
  worker diagnostics, and repo local launcher to expose and use the new gate.

## What was verified

- `go test ./...` passed in `services/agent-runtime`
- `uv run pytest tests/test_agent_gateway_outbox_worker.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q` passed
- Live runtime verification on `127.0.0.1:8780` confirmed:
  - `outbox_execution_scope=account_conversation_kind_gated`
  - first-account private file outbox event
    `qq-account-conv-gate:first-private-file:c65f6cb5bcbe4e589fdcdc396194c66b`
    auto-succeeded through Go local worker
  - first-account group file outbox event
    `qq-account-conv-gate:first-group-file:fac830d6a62847a997b2521e899b259d`
    stayed `queued/attempts=0`
  - second-account group file outbox event
    `qq-account-conv-gate:second-group-file:e2681d140af248dd91b6191412b201bc`
    still auto-succeeded

## Outcome

Go default execution ownership can now safely cover:

- all text
- second-account group/private file
- first-account private file

It still cannot safely claim:

- any image
- first-account group file

The goal remains active because Telegram backend is still missing token-backed
smoke, and QQ rich-media image / first-account group-file blockers still exist.
