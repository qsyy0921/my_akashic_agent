# SPEC-139 QQ Group Media Live Smoke Runbook

## Problem

The repo already has:

- read-only QQ cutover gates;
- a private-text live smoke script;
- real observe-only QQ group targets synced into Go.

What is still missing is a repeatable repo-owned runbook for:

- one real QQ group text send;
- one real QQ group image send;
- one real QQ group file send;
- explicit capture of whether media cases succeed or are blocked by the live
  NapCat / QQ platform session.

During this work, the OneBot WebSocket adapter must also tolerate unrelated
non-echo event frames whose `message` field is not always a JSON string.

## Goal

Add a repo-owned local smoke script that:

1. reads runtime config and delivery adapter health;
2. resolves one enabled observe-only QQ group target for a chosen account;
3. sends bounded text, image, and file group smoke deliveries through
   `POST /v1/outbound` plus `POST /v1/delivery-dispatch/send`;
4. marks only successful smoke deliveries `succeeded`;
5. records structured evidence for success and failure cases;
6. never fabricates Telegram readiness or full cutover success.

## Non-Goals

- Do not enable `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true`.
- Do not perform real outbound cutover.
- Do not move Python AI logic into Go.
- Do not hide platform rich-media failures behind synthetic success.

## Script Contract

The repo-owned entrypoint is:

```text
scripts/run-qq-group-live-smoke.ps1
```

Default scope:

- account `1049511700`
- channel alias `qq_1049511700`
- one enabled QQ observe-only group target, auto-resolved from
  `GET /v1/observe-targets`
- one text case
- one image case
- one file case

The script must:

1. call `GET /v1/runtime-config`;
2. call `GET /v1/delivery-adapters/health?timeout_seconds=5`;
3. call `GET /v1/observe-targets` and resolve a concrete group id;
4. create temporary local smoke assets for image/file cases;
5. call `POST /v1/outbound` for each case;
6. call `POST /v1/delivery-dispatch/readiness` before send;
7. call `POST /v1/delivery-dispatch/send`;
8. call `POST /v1/outbox/{event_id}/succeeded` only when dispatch succeeds and
   the operator did not opt out;
9. call `GET /v1/outbox/{event_id}` for every case;
10. emit JSON evidence for text/image/file cases.

## Adapter Invariant

The OneBot WebSocket adapter must ignore unrelated non-echo event frames even
when those frames carry a non-string `message` payload, such as a segment array
from a platform event.

The adapter still uses the echoed action response as the authoritative send
result.

## Acceptance

- Operators can run one repo-owned script and get structured evidence for QQ
  group text/image/file sends.
- Group text success is proven when the provider message id is returned and the
  smoke outbox record reaches `succeeded`.
- Group image/file outcomes are recorded as either real success or an explicit
  platform/runtime blocker, not guessed from readiness alone.
- The resulting documentation clearly separates:
  - successful group text smoke;
  - image/file platform blockers, if any;
  - the still-blocked outbox execution-owner cutover.
