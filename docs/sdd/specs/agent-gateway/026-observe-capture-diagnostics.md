# SPEC-026: Observe Capture Diagnostics

## Status

Implemented as a Go-owned read-only diagnostics slice.

## Context

QQ observe-only groups are now configured and synced into Go as observe targets.
Python still owns platform SDK handling, media download, and VLM/OCR summaries,
while Go owns the deterministic runtime stores: receiver status, inbox events,
media asset registry, and safe content access.

Operators need one authoritative place to answer whether an observe-only group
has actually produced:

- text events;
- image assets;
- file assets;
- locally readable media content for dashboard preview.

Without this diagnostic, a group can appear configured while silently missing
image/file capture or safe media access.

## Decision

Go exposes:

```text
GET /v1/observe-capture-diagnostics
```

The endpoint joins Go-owned state:

- observe targets from `/v1/observe-targets`;
- receiver lifecycle from `/v1/receiver-statuses`;
- raw observed messages from the inbox store;
- media assets from the media registry;
- bounded local media content probes via the existing safe media reader.

The result is read-only and reports `side_effect=none`. Content probes only open
bounded local media files through the existing safe-root policy; they do not call
QQ, Telegram, RAGFlow, VLM, OCR, or model APIs.

## Contract

Each target reports:

- target/channel identity;
- receiver connected state;
- inbox event count and text event count;
- attachment event/count;
- image/file media asset counts;
- media content ready/unavailable/forbidden/disabled counts;
- coverage booleans for text, attachment, image, file, and media content;
- status: `ok`, `warn`, `danger`, or `muted`;
- blockers such as `receiver_not_connected`, `image_not_seen`, `file_not_seen`,
  or `media_content_not_fully_ready`.

Runtime overview includes:

- `observe_capture_targets`;
- `observe_capture_ready`;
- `observe_capture_warning`;
- `observe_capture_blocked`;
- `observe_capture_text`;
- `observe_capture_image`;
- `observe_capture_file`;
- `observe_capture_content_ready`;
- an `Observe Capture` card.

## Boundaries

Go owns:

- capture coverage calculation;
- receiver/inbox/media registry joins;
- safe local media content readiness probing;
- runtime overview aggregation.

Python owns:

- QQ/NapCat event callbacks;
- group file URL lookup and download;
- image/file summarization;
- OCR/VLM/model behavior;
- dashboard display normalization.

## Acceptance

- `GET /v1/observe-capture-diagnostics` returns target-level coverage and
  totals.
- Runtime overview and dashboard expose the observe capture summary/card.
- No observe-only reply behavior changes.
- No live platform sends are triggered.
- Tests cover Go service, HTTP endpoint, runtime overview aggregation, Python
  client, and dashboard normalization.
