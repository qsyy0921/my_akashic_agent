# 210. Unified Goal Verifier Current-State Artifact

## Problem

`scripts/verify-go-migration-goal.ps1` already aggregates the live runtime,
dashboard, cutover, scheduler, MQ, Telegram, and worker-status verifiers.
However, before this slice, the script only printed JSON to stdout. That left
two problems:

1. current-turn evidence could be lost or drift from older local artifacts;
2. operators had to manually dig through nested sections to answer the stable
   questions used in migration close-out reports.

This is not a new runtime feature. It is a control-plane verification contract
for keeping the canonical live migration snapshot reproducible on disk.

## Scope

- Keep Go/Python ownership unchanged.
- Keep all existing live probes, cutover checks, and dashboard parity checks.
- Add a deterministic current-state snapshot and artifact write path to the
  unified goal verifier.

## Non-Goals

- Do not change runtime cutover policy.
- Do not enable Telegram, QQ group send, external lease, or any executor.
- Do not add new dashboard control actions.

## Required Behavior

### 1. Canonical artifact

`scripts/verify-go-migration-goal.ps1` must write its full JSON result to a
repo-owned artifact file:

- default path: `E:\agent\my-akashic_agent\.codex-goal-verifier.json`
- optional override: `-JsonOutputPath <path>`

The script must still print the same JSON to stdout.

### 2. Stable current-state snapshot

The unified JSON must expose a top-level `current_state` object containing the
fields required by migration status reporting:

- `qq_group_send_enabled`
- `telegram_token_configured`
- `outbox_execution_owner`
- `outbox_execution_scope`
- `agent_job_execution_owner`
- `agent_job_ack_owner`
- `agent_job_external_lease_ready`
- `agent_job_external_lease_decision`
- `receiver_statuses_telegram`
- `agent_workers_total`
- `agent_workers_stale`
- `runtime_overview_agent_workers_stale`
- `dashboard_fallback_category`

These fields are a normalized summary over already-existing live probes. They
must not introduce a second source of truth.

### 3. No ownership drift

The verifier may summarize:

- Go control-plane state
- Python-owned execution boundaries
- dashboard parity evidence

It must not mutate runtime configuration, enable workers, publish MQ work, send
platform messages, or execute Python AI logic.

## Acceptance

- Running `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
  writes the current-turn JSON to `.codex-goal-verifier.json`.
- The written JSON contains `current_state` with the required normalized fields.
- Existing residual classification and dashboard read-model evidence remain
  present.
