# SPEC-107: Media Retention Control Mutation Policy

## Status

Accepted for the current iteration.

## Context

`GET /v1/media-assets/retention-plan` now recommends operator approval and a
planned control mutation audit for future cleanup. However, the Go-owned control
mutation policy allowlist does not yet recognize
`target_kind=media_asset_retention` with `action=cleanup_expired`.

Without this policy entry, a future operator preflight would reject the exact
target/action emitted by the retention plan. The policy should recognize the
intent, while still keeping cleanup execution out of scope.

## Boundary Analysis

Go owns:

- deterministic control mutation target/action allowlist;
- preflight validation before any future control-plane mutation;
- audit-friendly policy visibility.

Python owns:

- OCR/VLM/file parsing/semantic extraction;
- no role in control mutation allowlist decisions.

Out of scope:

- deleting media registry rows;
- deleting local media files;
- creating approvals or mutation audits automatically;
- running any cleanup executor.

## Decision

Add `media_asset_retention` with action `cleanup_expired` to the domain
`ControlMutationPolicy`.

The preflight service remains unchanged except that the new target/action will
pass policy validation when a matching active approval exists. The suggested
audit remains `status=planned`; no mutation is executed.

## Acceptance

- Policy API lists `media_asset_retention/cleanup_expired`.
- Preflight allows `media_asset_retention cleanup_expired` with matching active
  approval.
- Unsupported media retention actions remain blocked.
- Full Go regression passes.
- TODO is cleared at the end of the iteration.
