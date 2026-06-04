# 211. Unified Goal Verifier Migration Residuals

## Problem

`scripts/verify-go-migration-goal.ps1` already gathers the live evidence needed
to classify the remaining Go migration work. Before this slice, that
classification still had to be reconstructed manually from nested verifier
output.

The migration close-out loop repeatedly needs the same eight classifications:

- QQ / NapCat cutover
- Telegram backend
- AgentJob external lease result-ack
- Scheduler runtime
- Proactive / worker control executors
- Media recovery private-source executor
- Dashboard read-model cleanup
- MQ adapter boundary

Without a stable machine-readable summary, each turn risks drifting back to
manual interpretation instead of reusing current-turn evidence.

## Scope

- Extend the unified goal verifier output with a top-level
  `migration_residuals` object.
- Each residual entry must summarize current category, key live facts, and the
  next minimal step.

## Non-Goals

- Do not change runtime cutover gates or ownership.
- Do not enable QQ group send, Telegram, external lease, or worker-control
  executors.
- Do not add new runtime-overview UI controls.

## Required Behavior

### 1. Canonical residual entries

The verifier must emit these top-level entries under `migration_residuals`:

- `qq_napcat_cutover`
- `telegram_backend`
- `agent_job_external_lease_result_ack`
- `scheduler_runtime`
- `worker_control_executors`
- `media_recovery_private_source_executor`
- `dashboard_read_models`
- `mq_adapter_boundary`

### 2. Evidence-backed fields

Each residual entry must be derived from already collected live evidence and
must expose:

- a stable `category`
- current key facts for that area
- `next_minimal_step`

### 3. No second source of truth

`migration_residuals` is a normalized reporting layer over existing live
verifiers. It must not introduce speculative state or replace the underlying
verifier outputs.

## Acceptance

- Running the unified verifier writes `.codex-goal-verifier.json`.
- That artifact contains `migration_residuals` with the eight required entries.
- Each entry is populated from current-turn live evidence rather than hardcoded
  prose alone.
