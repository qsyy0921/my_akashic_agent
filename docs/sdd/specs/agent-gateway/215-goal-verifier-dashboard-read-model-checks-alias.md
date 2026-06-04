# SDD: Goal Verifier Dashboard Read-Model Checks Alias

## Problem

`scripts/verify-go-migration-goal.ps1` already emits the current-turn
`dashboard_read_models` evidence at top level, but some consumers probe
`checks.dashboard_read_models.*`.

That mismatch does not change live verification truth, but it weakens the
canonical artifact contract and produces false `null` reads for otherwise
verified dashboard evidence.

## Non-goals

- Do not change any runtime endpoint, worker, queue owner, or dashboard behavior.
- Do not remove the existing top-level `dashboard_read_models` contract.
- Do not add new dashboard drilldowns.

## Requirements

1. The unified verifier must continue exposing top-level
   `dashboard_read_models`.
2. The unified verifier must also expose the same object under
   `checks.dashboard_read_models`.
3. The alias must preserve the current proactive dashboard fallback evidence,
   including `proactive_tick_logs_readable`.
4. Focused tests must pin the alias so future refactors do not silently remove
   it.

## Invariants

- No live endpoint semantics change.
- No QQ, Telegram, MQ, scheduler, AgentJob, or worker mutation is introduced.
- The alias is evidence-shape hardening only.

## Acceptance

- `.codex-goal-verifier.json` contains both:
  - `dashboard_read_models.*`
  - `checks.dashboard_read_models.*`
- `proactive_tick_logs_readable` is readable from the aliased path after
  rerunning `scripts/verify-go-migration-goal.ps1`.
