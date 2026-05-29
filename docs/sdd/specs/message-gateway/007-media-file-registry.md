# SPEC-007: Media And File Registry

## Status

Draft for review before implementation.

## Context

QQ group observation already captures image and file metadata in the Python
compatibility layer. The dashboard can show some attachment links, but the
target architecture requires Go to own stable asset ids, retention policy, and
controlled dashboard access.

Raw platform URLs, CQ codes, or local file paths should not become the durable
contract between QQ/NapCat, Python agent logic, and dashboard UI.

## Decision

Add a Go-owned `MediaAsset` aggregate and registry API.

The registry stores metadata first:

- `asset_id`
- platform and account id
- conversation id and conversation type
- source message id
- sender id
- kind: `image`, `file`, `audio`, `video`, or `unknown`
- original URL or local path reference
- MIME type, file name, and size
- optional content hash
- retention policy
- created and updated timestamps
- metadata map

Binary download and OCR/vision summaries are later workers. The first registry
slice must not block inbound message ingestion on network downloads.

## HTTP Contract

Register asset metadata:

```text
POST /v1/media-assets
```

Query recent assets:

```text
GET /v1/media-assets?limit=50
```

Query one asset:

```text
GET /v1/media-assets/{asset_id}
```

Future controlled bytes endpoint:

```text
GET /v1/media-assets/{asset_id}/content
```

The first implementation may return metadata only and reject `/content` unless
a local safe path has been registered and access policy is implemented.

## Asset ID

Asset ids are stable and account-aware:

```text
asset:{platform}:{account_id}:{conversation_type}:{conversation_id}:{source_message_id}:{index}
```

When a platform provides a strong id, it may be stored in metadata, but the
Akashic asset id remains the primary contract.

## Safety Rules

- Dashboard must not expose arbitrary local paths as direct links.
- Content routes must reject path traversal and paths outside the configured
  workspace asset directory.
- Observe-only groups can register assets but must not trigger group replies.
- Python vision/RAG workers consume asset ids and request bytes through Go
  routes after content access policy exists.
- Account id is mandatory in every media asset.

## Acceptance Tests

- Valid asset registration returns a stable asset id.
- Duplicate asset id registration is idempotent.
- Listing returns newest assets first.
- Asset query includes account id, conversation id, kind, file name, and source
  message id.
- Content route is not enabled before access policy is implemented.

## Migration Plan

1. Add Go domain/app/API/media registry with in-memory store.
2. Add JSONL or SQLite persistence.
3. Mirror Python-captured QQ attachments to `/v1/media-assets` in shadow mode.
4. Add dashboard panel/link rendering through Go metadata.
5. Add controlled bytes endpoint.
6. Pass asset ids into Python vision/RAG workers.
