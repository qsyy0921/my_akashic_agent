# Review: Phase 8.64 Outbox Dashboard Readiness

Spec:
`docs/sdd/specs/agent-gateway/015-delivery-dispatch-plan.md`

Implementation summary:
- Outbox dashboard detail now calls Go `POST /v1/delivery-dispatch/readiness`
  after loading the delivery record.
- The diagnostic request passes `AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT` so
  account-specific QQ/NapCat aliases can be checked before cutover.
- The frontend renders adapter readiness, missing channels, side effect, and
  the dispatch plan in the outbox detail panel.
- Readiness failures are reported inside `dispatch_readiness` and do not break
  the existing outbox detail page.

Tests run:
- `uv run pytest tests\test_outbox_dashboard_plugin.py -q --basetemp .tmp\pytest-outbox-dashboard-readiness`
- `uv run pytest tests\test_outbox_dashboard_plugin.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-outbox-dashboard-readiness-regression`
- `npm run typecheck`
- `npm run build:plugins`
- `git diff --check`

Findings:
- This is read-only dashboard plumbing. It does not call dispatch, acknowledge
  queue messages, or send QQ/Telegram traffic.
- The readiness endpoint is best-effort from the dashboard perspective, so older
  or temporarily unavailable runtimes still allow delivery inspection.

Decision:
Accept the slice as a safe pre-cutover diagnostic improvement for QQ/NapCat
adapter configuration.

Follow-ups:
- Configure `AKASHIC_ONEBOT_WS_URLS` and `AKASHIC_ONEBOT_ACCESS_TOKENS`, then
  verify the dashboard shows QQ channel aliases as ready.
- Run explicit QQ/NapCat live-send smoke only after user confirmation.
