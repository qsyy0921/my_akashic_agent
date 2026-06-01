# SPEC-135: Runtime Overview Media Content Recovery

## Status

Accepted for the current iteration.

## Context

`SPEC-134` added an approval-bound media content recovery executor. Applied and
failed recoveries are now recorded as Go-owned control mutation audits with:

- `target_kind=media_asset_content`
- `action=recover_content`

Operators currently need to inspect the generic control audit detail to see
whether media recovery is working. Runtime overview should expose a focused
read-only recovery card without adding another executor path or invoking Python
AI.

## Boundary Analysis

Go owns:

- filtering control mutation audit records for media content recovery;
- summary/card/detail fields in runtime overview;
- stable endpoints metadata for plan/preflight/executor.

Python owns:

- OCR/VLM/file parsing after content becomes available;
- semantic memory/RAG enrichment triggered by future AgentJob designs;
- provider-specific private media fetch strategies.

Out of scope:

- executing content recovery from runtime overview;
- creating approvals or mutation audits;
- calling `/v1/media-assets/content-recovery`;
- enqueueing OCR/VLM/RAG jobs.

## Decision

Add `MediaAssetContentRecoveryOverviewView` to runtime overview:

- `ready`: true when the overview source is available;
- `reason`;
- `recent_audits`;
- `totals` for `audits`, `planned`, `applied`, `failed`, `rolled_back`;
- `endpoints` for `plan`, `preflight`, and `recovery`;
- `side_effect=none`.

Runtime overview summary adds:

- `media_asset_content_recovery_ready`;
- `media_asset_content_recovery_reason`;
- `media_asset_content_recovery_applied`;
- `media_asset_content_recovery_failed`;
- `media_asset_content_recovery_recent_audits`.

Runtime overview cards add:

- ID: `media_asset_content_recovery`
- Label: `Media Content Recovery`

## Acceptance

- Runtime overview includes a focused `media_asset_content_recovery` detail.
- The card is `danger` when failed audits exist, `ok` when applied audits exist,
  otherwise `muted`.
- The overview remains read-only and does not create approvals, audits, jobs, or
  downloads.
- Go runtime tests cover summary/card/detail.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/PROJECT_STATUS/review/index are updated.
- TODO is cleared after verification.
