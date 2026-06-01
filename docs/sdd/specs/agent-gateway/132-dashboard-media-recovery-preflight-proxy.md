# SPEC-132: Dashboard Media Recovery Preflight Proxy

## Status

Accepted for the current iteration.

## Context

Go now exposes `/v1/media-assets/content-recovery/preflight` as a read-only,
approval-bound preflight for future media content recovery/download executors.
The dashboard already proxies content access and recovery plan endpoints, and
message media entries expose content/access/recovery-plan URLs. Without a
dashboard preflight proxy and URL, operators still need to manually call the Go
endpoint.

## Boundary Analysis

Go owns:

- media content recovery plan and preflight semantics;
- operator approval check and control mutation preflight;
- media recovery executor boundaries and future audit binding.

Python dashboard owns:

- read-only proxying to Go runtime;
- deterministic dashboard URL enrichment for media asset rows.

Out of scope:

- creating approvals or mutation audits;
- downloading, restoring, caching or streaming media content;
- OCR/VLM/RAG/AI work;
- changing Go recovery policy from Python.

## Decision

Add dashboard proxy:

- `GET /api/dashboard/media-assets/content-recovery/preflight`

It forwards query parameters to Go:

- `asset_id`
- optional `target_id`
- optional `operator_id`
- optional `approval_id`

Message media assets gain:

- `content_recovery_preflight_url`

The URL includes only `asset_id` by default. Operators provide approval-specific
query parameters when they explicitly inspect or execute a preflight. List/detail
rendering must not call the preflight endpoint.

## Acceptance

- Dashboard proxy returns valid Go preflight payloads.
- Invalid Go responses return 502 with a clear message.
- Message media entries include `content_recovery_preflight_url`.
- Tests prove proxy query forwarding and URL enrichment.
- SDD DONE/LIVE_CHECKS/OPEN_ISSUES/review/index are updated.
- TODO is cleared after verification.
