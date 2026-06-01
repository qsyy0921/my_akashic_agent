# SPEC-131: Media Content Recovery Preflight

## Status

Accepted for the current iteration.

## Context

Go already owns media asset metadata, content access diagnostics, content access
plans, and read-only content recovery plans. OI-005 remains open because a
future executor for restoring or re-downloading missing media content still
needs a safe approval/audit boundary before any destructive or external side
effect is introduced.

This slice adds that boundary as a read-only preflight. It does not implement
the executor.

## Boundary Analysis

Go owns:

- media asset metadata and deterministic content access/recovery plans;
- control mutation policy allowlist;
- operator approval check and control mutation preflight;
- a read-only media content recovery preflight API.

Python owns:

- OCR/VLM/file parsing;
- semantic interpretation of media content;
- AI-driven tool selection and recovery strategy experiments.

Out of scope:

- downloading media from QQ/Telegram/other platforms;
- restoring files, writing cache content, or modifying media metadata;
- creating operator approvals or control mutation audits;
- running OCR/VLM/RAG/AI work.

## Decision

Add a control mutation policy intent:

- `target_kind=media_asset_content`
- `action=recover_content`

Add `GET /v1/media-assets/content-recovery/preflight` with query parameters:

- `asset_id`
- optional `target_id` (defaults to `asset_id`)
- `operator_id`
- `approval_id`

The preflight:

1. Reads the existing content recovery plan.
2. Blocks if recovery is not required or not actionable.
3. Runs Go control mutation preflight for `media_asset_content/recover_content`.
4. Returns a suggested planned audit when approval is active.

The endpoint has `side_effect=none`; it does not create approval/mutation
records and does not download, restore, stream, parse, or invoke AI.

## Acceptance

- Control mutation policy lists `media_asset_content/recover_content`.
- Content recovery preflight returns ready only when recovery is needed and the
  approval-bound control preflight is ready.
- Missing approval or no recovery candidate returns stable blockers.
- HTTP handler accepts GET only and returns a stable JSON result.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
