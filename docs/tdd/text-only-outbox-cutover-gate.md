# TDD: Text-only Outbox Cutover Gate

## Targeted Coverage

1. Python compatibility outbox worker:
   - backs off when Go runtime reports
     `workers.outbox_delivery_worker_enabled=true`
   - does not lease or dispatch deliveries while Go owns outbox execution
2. Repo launcher:
   - `scripts/start-agent-runtime.ps1` can propagate
     `AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS`
3. Manual live verification:
   - real QQ private text outbox event is auto-sent by Go local worker
   - real QQ image/file outbox events remain queued in text-only mode
   - knowledge planner remains healthy after the cutover change

## Commands

- `uv run pytest tests/test_agent_gateway_outbox_worker.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Out of Scope

- Automated PowerShell tests for the repo launcher
- Automated NapCat / QQ rich-media success tests
- Telegram backend smoke without a real bot token
