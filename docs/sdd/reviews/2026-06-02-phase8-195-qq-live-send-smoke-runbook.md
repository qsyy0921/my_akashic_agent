# Phase 8 Review 195: QQ Live Send Smoke Runbook

## Spec

- `docs/sdd/specs/agent-gateway/138-qq-live-send-smoke-runbook.md`

## Scope

- Added a repo-owned PowerShell runbook for QQ private-text live send smoke.
- Kept the smoke path on top of existing Go runtime APIs instead of enabling
  full Go outbox worker execution.

## Boundary Review

- The script uses Go-owned outbox and dispatch endpoints only for explicit
  smoke event ids.
- It does not enable `integrations.agent_runtime.outbound_channels` for QQ.
- It does not enable Go local outbox worker or NATS external lease.
- Telegram remains out of scope and must stay blocked when its token is absent.

## Verification

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...` from `services/agent-runtime`
- `uv run pytest tests/test_agent_gateway_client.py -q`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `.\scripts\run-qq-live-smoke.ps1 -SkipLogCheck`

## Findings

- The new repo-owned smoke entrypoint successfully exercised real OneBot private
  text sends in both directions:
  - `1049511700 -> 2365524513`
  - `2365524513 -> 1049511700`
- Both dispatches returned `status=sent` with non-empty OneBot
  `provider_message_id`, and the script marked both smoke outbox events
  `succeeded`.
- `log_observed=false` is acceptable for the default script mode because the
  local QQ config intentionally treats bot-peer private messages without trigger
  prefixes as non-interactive traffic; the smoke relies on provider acceptance
  and outbox state instead of AI-side reply handling.

## Residual Risk

- Private-text smoke does not prove group text, image, or file upload behavior.
- Optional log confirmation is best-effort because bot-peer trigger policy may
  intentionally ignore unprefixed bot-to-bot private content.
