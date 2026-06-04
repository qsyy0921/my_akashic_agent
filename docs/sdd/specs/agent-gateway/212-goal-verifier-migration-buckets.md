# 212. Unified Goal Verifier Migration Buckets

## Problem

The unified verifier now emits machine-readable residual entries, but the goal
requires every remaining area to be classified in one of four exact buckets:

- already migrated to Go, only missing live verification
- Go control plane exists, but the real executor is missing
- cutover still incomplete
- explicitly retained in Python

Without an exact normalized bucket layer, final reporting still needs a manual
translation step.

## Scope

- Extend the unified goal verifier artifact with exact migration bucket fields.
- Add a bucketed top-level summary over the residual entries.

## Non-Goals

- Do not change the underlying residual evidence.
- Do not change runtime ownership or cutover gates.

## Required Behavior

### 1. Per-residual bucket

Each `migration_residuals.*` entry must expose `migration_bucket` using one of
these exact values:

- `already_in_go_only_missing_live_verification`
- `go_control_plane_present_but_real_executor_missing`
- `still_not_fully_cut_over`
- `explicitly_python_owned`

### 2. Top-level summary

The artifact must expose `migration_bucket_summary` with one array per exact
bucket. Each array should list the residual item names currently in that
bucket.

### 3. Python-owned surfaces

The artifact must include an explicit residual entry for Python-owned surfaces
to make the non-migration boundary machine-readable.

## Acceptance

- Running the unified verifier writes `.codex-goal-verifier.json`.
- That artifact contains both per-entry `migration_bucket` and a top-level
  `migration_bucket_summary`.
- The summary is derived from current residual entries rather than handwritten
  external prose.
