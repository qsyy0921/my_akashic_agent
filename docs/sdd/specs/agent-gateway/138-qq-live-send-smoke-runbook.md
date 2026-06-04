# SPEC-138 QQ Live Send Smoke Runbook

## Problem

The repo already has read-only QQ/NapCat cutover gates, but operators still
need a reproducible local smoke path that proves Go can perform a real OneBot
private send without enabling the full Go outbox worker or changing Python's
AI/runtime ownership.

Today that path exists only as low-level HTTP endpoints:

- `POST /v1/outbound`
- `POST /v1/delivery-dispatch/readiness`
- `POST /v1/delivery-dispatch/send`
- `POST /v1/outbox/{event_id}/succeeded`

That is enough for manual experimentation, but not enough for repeatable
operator verification.

## Goal

Add a repo-owned local smoke script that:

1. checks runtime config and delivery adapter health;
2. creates a bounded QQ private-text outbox delivery for each bot direction;
3. dispatches it through Go `delivery-dispatch/send`;
4. marks the smoke delivery `succeeded` after successful dispatch so the local
   outbox state is not left queued;
5. records evidence in machine-readable output.

## Non-Goals

- Do not enable `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`.
- Do not add QQ aliases to `integrations.agent_runtime.outbound_channels`.
- Do not treat private-text smoke as proof of group/image/file behavior.
- Do not fabricate Telegram backend success or modify Telegram credentials.
- Do not move Python AI execution logic into Go.

## Script Contract

The repo-owned entrypoint is:

```text
scripts/run-qq-live-smoke.ps1
```

Default scope:

- two private text sends:
  - `1049511700 -> 2365524513`
  - `2365524513 -> 1049511700`
- runtime base URL `http://127.0.0.1:8780`
- channel aliases:
  - `1049511700 -> qq_1049511700`
  - `2365524513 -> qq_2365524513`

The script must:

1. call `GET /v1/runtime-config`;
2. call `GET /v1/delivery-adapters/health?timeout_seconds=5`;
3. create unique smoke messages and event ids;
4. call `POST /v1/outbound` for each direction;
5. call `POST /v1/delivery-dispatch/readiness` before send and fail if
   `ready=false`;
6. call `POST /v1/delivery-dispatch/send`;
7. call `POST /v1/outbox/{event_id}/succeeded` unless explicitly skipped;
8. call `GET /v1/outbox/{event_id}` to record final state;
9. emit JSON evidence for both directions.

## Safety Rules

- The default smoke content is a unique ASCII token and does not include QQ
  peer trigger prefixes.
- This default avoids intentionally triggering AI reply behavior from the peer
  bot account.
- Optional log verification is best-effort only because unprefixed bot-to-bot
  private messages may be intentionally ignored by the configured peer-trigger
  policy.
- The script is allowed to mutate only the local outbox lifecycle for its own
  smoke event ids.

## Acceptance

- Operators can run one script from the repo root and get structured evidence
  for a real QQ private-text send attempt in both directions.
- Successful smoke proves real Go OneBot private text dispatch for the two
  configured local QQ accounts without enabling the full Go outbox worker.
- `LIVE_CHECKS.md` and `OPEN_ISSUES.md` clearly distinguish:
  - verified private-text smoke;
  - unverified group/image/file smoke;
  - still-blocked cutover execution owner work.
