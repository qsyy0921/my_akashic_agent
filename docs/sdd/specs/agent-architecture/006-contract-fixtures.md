# SPEC-006: Contract Fixtures

## Status

Draft

## Context

The Go/Python split cannot be implemented safely until both runtimes agree on
event and job contracts. This spec defines the fixture set that must exist
before migration work.

## Fixture Directory

Use:

```text
tests/fixtures/contracts/
```

Fixtures should be plain JSON and stable enough to be loaded by both Go and
Python tests.

## Required Fixtures

| Fixture | Purpose |
| --- | --- |
| `message_envelope.qq.private.text.json` | QQ private text normalization |
| `message_envelope.qq.group.image.json` | QQ group image event with asset placeholder |
| `agent_inbound.allowed.private.json` | event allowed into Python agent runtime |
| `agent_inbound.observe_only.group.json` | observed group event that cannot reply |
| `agent_decision.reply.private.json` | Python reply routed through Go outbox |
| `agent_decision.create_image_job.json` | Python creates long-running image job |
| `media_asset.qq.image.json` | image asset metadata and local/signed URL |
| `media_asset.qq.file.json` | file asset metadata |
| `image_job.lifecycle.json` | pending/running/succeeded/failed image job states |
| `memory_extract_job.group_thread.json` | group memory extraction job |
| `rag_ingest_job.thread_summary.json` | RAG indexing job for thread summary |
| `rag_eval_job.group_memory.json` | Go-owned RAG evaluation job executed by Python eval worker |
| `knowledge_checkpoint.ragflow.qq.json` | Go-owned RAGFlow checkpoint cursor committed by Python worker; `cursor` matches the runtime integer API and extended cursor details live in metadata |
| `inbox_replay.observe_only.qq.json` | Go inbox replay window for observe-only group memory extraction |
| `outbox_delivery.qq.private.text.json` | Go outbox delivery state, OneBot adapter result, and echo-loop guard |
| `media_asset_content.qq.image.json` | Go media content access contract for dashboard previews/downloads |
| `agent_job_event_stream.rag_ingest.json` | Go durable job lifecycle event stream for RAG ingest jobs |
| `group_thread.hardware.json` | segmented hardware discussion |
| `group_thread.game_guide.json` | segmented game攻略 discussion |
| `source_citation.message_asset.json` | citation linking answer to message and asset |

## Contract Requirements

Every fixture must include:

- `schema_version`
- stable `event_id` or `job_id`
- `platform`
- `account_id`
- `conversation_id`
- `conversation_type`
- `agent_id` when routed to an agent
- source ids for derived records
- timestamps with timezone
- metadata map for forward-compatible fields

## Compatibility Tests

Before implementation:

- Go test loads every fixture and validates domain/application DTOs.
- Python test loads every fixture into typed dataclasses.
- Unknown metadata fields are preserved.
- Dashboard/worker smoke tests consume the same fixtures for checkpoint lag,
  media content proxying, RAG worker cursor reads, and job event stream reads.
- Required fields fail fast with clear error messages.
- Round-trip serialization preserves ids and source refs.

## Acceptance

No Go adapter migration, Python bus replacement, media registry implementation,
or group-memory worker implementation can start until these fixtures are written
and loaded by both runtimes.
