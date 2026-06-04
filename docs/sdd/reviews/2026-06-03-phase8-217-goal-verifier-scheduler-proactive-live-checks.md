# Review: goal verifier scheduler proactive live checks

Spec:
- `docs/sdd/specs/agent-gateway/160-goal-verifier-scheduler-proactive-live-checks.md`

Implementation summary:
- Extended `scripts/verify-go-migration-goal.ps1` with live scheduler
  verification based on `/v1/scheduler/jobs`, `/v1/scheduler/leases`, and
  `/v1/scheduler/diagnostics`.
- Extended the unified verifier with live proactive verification based on
  `/v1/proactive/tick-logs`, drift summary, anyaction quota, bg-context,
  context-only, and dashboard `/api/dashboard/proactive/tick_logs`.
- Replaced `not_current_turn` residual placeholders for scheduler,
  proactive, dashboard fallback, and agent_job external lease result-ack with
  current-turn classifications.

Tests run:
- `.\scripts\verify-go-migration-goal.ps1`
- `.\scripts\verify-go-migration-goal.ps1 -IncludeOutboxScopeSmoke`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

Findings:
- Scheduler is now classified from live runtime evidence, and the current local
  runtime is provably idle rather than “unverified”.
- Proactive is now classified from live runtime and dashboard evidence; the
  current local runtime has recent Go-owned tick logs, visible anyaction quota,
  and readable dashboard fallback.
- AgentJob external lease result-ack remains blocked, but the blocker is now
  explicitly current-turn evidence instead of a placeholder.

Decision:
- Accept.

Follow-ups:
- Keep reminder/recurring/recovery and proactive flow-specific observations in
  `LIVE_CHECKS.md`; they are still real remaining work, but no longer justify
  `not_current_turn` placeholders in the unified goal verifier.
