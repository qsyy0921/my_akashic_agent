# SPEC-007: Media And File Registry

## Status

Implemented through metadata registry, controlled local content access, and
file-backed registry persistence in Go `agent-runtime`.

## Context

QQ group observation already captures image and file metadata in the Python
compatibility layer. The dashboard can show some attachment links, but the
target architecture requires Go to own stable asset ids, retention policy, and
controlled dashboard access.

Raw platform URLs, CQ codes, or local file paths should not become the durable
contract between QQ/NapCat, Python agent logic, and dashboard UI.

After the content route is enabled, observed message attachments must be
registered by the Go runtime during message ingest. Otherwise the dashboard can
observe attachments in `/v1/shadow/observed`, but it cannot reliably resolve the
same file through the Go media registry.

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

Message ingest registers every envelope attachment as a `MediaAsset`:

- source message id is the envelope `event_id`
- sender id is the envelope sender id
- channel route is copied from the envelope
- attachment id is used as the stable asset id when present
- generated asset id is used only when the platform/compat layer did not supply
  an id
- metadata records `registered_from=message_ingest`

Registration is idempotent. Existing asset ids are not overwritten by later
duplicate message delivery.

The media registry repository can be backed by a durable JSON file. Configure
`AKASHIC_MEDIA_ASSETS_DSN` or `AKASHIC_MEDIA_ASSETS_PATH`; the special value
`memory` keeps development-only in-memory behavior. Runtime startup must pass
the same repository to message ingest and media API services so shadow
registration, manual registration, query, and content access share one
authoritative registry.

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

Controlled bytes endpoint:

```text
GET /v1/media-assets/{asset_id}/content
```

The content route may serve only registered local files whose resolved path is
inside configured media roots. Remote URLs, path traversal, directories, missing
files, and files outside allowed roots are rejected.

Allowed roots are configured by `AKASHIC_MEDIA_ASSET_ROOTS` as a comma-separated
list. If omitted, the local runtime defaults to Akashic workspace upload
directories under the repository root.

Dashboard must use same-origin links and treat Go as the authority for media
bytes:

```text
GET /api/dashboard/media-assets/content?asset_id={asset_id}
```

The Python dashboard endpoint is only a thin proxy to:

```text
GET /v1/media-assets/{asset_id}/content
```

It must not serve arbitrary attachment paths for Go-registered assets, must not
fetch remote platform URLs directly, and must preserve the upstream content type
and content disposition when Go returns bytes. Shadow audit attachment metadata
adds `content_url` when an `asset_id` is present, and dashboard panels should
prefer `content_url` over raw `url`.

## Asset ID

Asset ids are stable and account-aware:

```text
asset:{platform}:{account_id}:{conversation_type}:{conversation_id}:{source_message_id}:{index}
```

When a platform provides a strong id, it may be stored in metadata, but the
Akashic asset id remains the primary contract.

## Safety Rules

- Dashboard must not expose arbitrary local paths as direct links.
- Content routes must reject path traversal and paths outside configured media
  roots after absolute path resolution.
- Content routes must serve files with registered or detected MIME type and an
  inline `Content-Disposition` so dashboard links can preview images directly.
- Content routes must not fetch remote URLs; download workers can mirror remote
  platform URLs into safe local roots before registration.
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
- Content route returns bytes for safe local files.
- Content route returns `403` for paths outside allowed roots.
- Content route returns `404` for missing or unsupported local content.
- Shadow message ingest automatically registers attachments into the media
  registry.
- File-backed media registry survives `agent-runtime` restart.
- Shadow audit dashboard emits same-origin `content_url` links for asset ids.
- Dashboard media proxy forwards bytes from Go content routes and maps upstream
  content errors without exposing local file paths.

## Migration Plan

1. Add Go domain/app/API/media registry with in-memory store.
2. Add JSON file persistence.
3. Mirror Python-captured QQ attachments to `/v1/media-assets` in shadow mode.
4. Add controlled bytes endpoint.
5. Add dashboard panel/link rendering through Go metadata.
6. Pass asset ids into Python vision/RAG workers.
