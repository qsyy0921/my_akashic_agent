# SPEC-002: Group Memory Model

## Status

Draft

## Context

Group memory should not be a single vector database. Group chats contain raw
events, social context, temporal decisions, files, screenshots, changing facts,
and reusable procedures. A single flat RAG index loses too much structure.

## Memory Layers

```text
L0 Raw Log          Go-owned immutable messages and assets
L1 Conversation     Python-built threads and topic windows
L2 Episodic         summaries of what happened in a time/thread window
L3 Semantic         stable facts, FAQs, product/game entities, claims
L4 Procedural       reusable攻略, troubleshooting flows, setup steps
L5 Social/Policy    group norms, trusted speakers, admin notices
L6 RAG Index        retrieval views over L1-L5 with citations
```

## Ownership

| Layer | Owner | Storage Shape |
| --- | --- | --- |
| L0 Raw Log | Go | append-only DB table or event store |
| L0 Assets | Go | asset registry with local path, mime, hash, source message |
| L1 Conversation | Python writes, Go can query source ids | thread records |
| L2 Episodic | Python | markdown/JSON summaries with message spans |
| L3 Semantic | Python | structured facts/entities/FAQs with validity metadata |
| L4 Procedural | Python | versioned procedure cards |
| L5 Social/Policy | Python | group profile and trust metadata |
| L6 RAG Index | Python | vector/sparse/graph indexes plus source refs |

## Core Entities

### GroupProfile

```json
{
  "group_id": "27234224",
  "name": "主机DIY交流群",
  "category": "hardware",
  "observe_only": true,
  "default_language": "zh-CN",
  "topic_taxonomy": ["cpu", "gpu", "motherboard", "psu", "case", "deal"],
  "reply_policy": "never_reply_without_owner_command"
}
```

### GroupFact

```json
{
  "fact_id": "fact:uuid",
  "group_id": "27234224",
  "subject": "7800X3D",
  "predicate": "recommended_for",
  "object": "gaming build",
  "claim": "7800X3D is commonly recommended for gaming-focused builds.",
  "valid_from": "2026-05-30T00:00:00+08:00",
  "valid_until": null,
  "confidence": 0.78,
  "source_message_ids": ["qqmsg:100", "qqmsg:103"],
  "source_asset_ids": [],
  "extraction_model": "mimo-vl-or-text-model",
  "status": "active"
}
```

### ProcedureMemory

```json
{
  "procedure_id": "proc:uuid",
  "group_id": "187890369",
  "domain": "game",
  "title": "洛克王国世界某任务打法",
  "preconditions": ["版本 ver9.1", "角色等级达到要求"],
  "steps": [
    {"order": 1, "text": "先完成前置任务"},
    {"order": 2, "text": "携带指定宠物或装备"}
  ],
  "warnings": ["版本更新后可能失效"],
  "source_thread_ids": ["thread:uuid"],
  "version": "2026-05-30",
  "confidence": 0.72
}
```

### ProductConfigMemory

```json
{
  "config_id": "config:uuid",
  "group_id": "27234224",
  "domain": "hardware",
  "budget": "5000-7000 CNY",
  "parts": [
    {"kind": "cpu", "name": "7800X3D"},
    {"kind": "motherboard", "name": "B650"}
  ],
  "rationale": "gaming performance and platform longevity",
  "tradeoffs": ["higher initial cost"],
  "source_thread_ids": ["thread:uuid"],
  "status": "active"
}
```

### AssetInsight

```json
{
  "asset_insight_id": "asset_insight:uuid",
  "asset_id": "asset:image:uuid",
  "group_id": "27234224",
  "source_message_id": "qqmsg:200",
  "asset_type": "image",
  "ocr_text": "BIOS version ...",
  "vision_summary": "screenshot of benchmark result",
  "extracted_entities": ["BIOS", "benchmark"],
  "safety_flags": [],
  "confidence": 0.8
}
```

## Validity and Conflict Handling

Group knowledge changes. Memory records need validity metadata:

- `valid_from`, `valid_until`
- `status`: `active`, `superseded`, `disputed`, `expired`, `deleted`
- `confidence`
- `source_count`
- `last_confirmed_at`
- `superseded_by`

When a new extraction contradicts an active fact:

1. Keep both records.
2. Mark conflict relation.
3. Prefer newer fact only when source confidence and recency are stronger.
4. During answer generation, surface uncertainty instead of hiding conflict.

## Trust Model

Group participants are not equally reliable. Do not hard-code "trusted users";
learn a soft trust signal:

- admin/moderator messages;
- repeated accepted answers;
- high source agreement;
- later corrections by other users;
- domain-specific reliability, such as hardware pricing vs game攻略.

Trust is a retrieval feature, not an absolute gate. Low-trust sources can still
appear with lower confidence.

## Retention

- Raw L0 messages: configurable retention; default keep for observed target
  groups during development.
- Assets: keep metadata permanently, bytes by retention policy.
- Summaries/facts/procedures: keep until superseded or manually deleted.
- RAG indexes: rebuildable from L0-L5; do not treat index as source of truth.

## Invariants

- Every memory record must cite at least one source message or source asset.
- Every generated answer about group knowledge must be traceable to memory
  records and raw messages.
- RAG index entries are derived views, not authoritative storage.
- Game攻略 and hardware recommendations must include version/time context when
  available.

