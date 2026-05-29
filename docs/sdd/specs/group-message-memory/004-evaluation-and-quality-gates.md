# SPEC-004: Evaluation And Quality Gates

## Status

Draft

## Context

Group memory and RAG can look impressive in demos while failing in production:
wrong source, outdated advice, hallucinated攻略, or answers based on noisy chat.
Implementation should not proceed to production behavior until the evaluation
gates are explicit.

## Evaluation Datasets

Use three dataset types:

| Dataset | Source | Purpose |
| --- | --- | --- |
| Synthetic QQ replay | generated from realistic group patterns | deterministic tests for pipeline behavior |
| Open-source dialogue/QA data | public Chinese conversation, forum, support, or community QA datasets | retrieval and answer baseline |
| Real observed group replay | local opted-in QQ logs from target groups | final shadow-mode validation |

Synthetic replay should cover:

- hardware build recommendation;
- game攻略 extraction;
- image screenshot analysis;
- file attachment summary;
- conflicting advice;
- outdated version-specific advice;
- noisy social chatter;
- unanswered question;
- later correction to earlier answer.

## Metrics

### Ingest and Segmentation

- duplicate drop rate;
- message loss rate;
- thread segmentation precision/recall on labeled replay;
- asset registration success rate;
- worker lag and queue age.

### Extraction

- fact precision and recall;
- procedure step accuracy;
- source citation completeness;
- contradiction detection recall;
- freshness/validity tagging accuracy;
- unsupported extraction rate.

### Retrieval

- Recall@k for source message/thread;
- MRR and nDCG for ranked source relevance;
- entity filter accuracy;
- asset-source retrieval accuracy;
- latency p50/p95.

### Answer Quality

- citation accuracy;
- faithfulness to cited source;
- conflict disclosure rate;
- answer usefulness judged by rubric;
- abstention quality when evidence is weak.

## Golden Tests

Each golden case should include:

```json
{
  "case_id": "hardware_conflict_001",
  "group_profile": {"group_id": "27234224", "category": "hardware"},
  "messages": [],
  "assets": [],
  "query": "这个群之前怎么评价 7800X3D 装机？",
  "expected_source_ids": ["qqmsg:10", "qqmsg:14"],
  "expected_memory_types": ["GroupFact", "ProductConfigMemory"],
  "must_include": ["时间", "来源", "不确定性"],
  "must_not_include": ["无来源断言"]
}
```

## Quality Gates Before Code Migration

### Gate A: Contract Gate

- Go and Python parse the same golden `MessageEnvelope`, `MediaAsset`,
  `AgentInboundEvent`, `MemoryExtractJob`, and `RagIngestJob` fixtures.
- Dashboard can render fixture messages/assets from typed data.

### Gate B: Shadow Ingest Gate

- Go receives mirrored QQ group events in shadow mode.
- No observed group outbound reply is emitted.
- Raw event count matches Python channel count for sampled windows.

### Gate C: Extraction Gate

- Synthetic replay produces source-cited facts/procedures.
- Every memory item has at least one source id.
- Failed extraction jobs are retryable and visible.

### Gate D: Retrieval Gate

- Golden retrieval cases meet minimum Recall@5 and MRR threshold.
- Image/file queries retrieve asset insights.
- Conflict cases retrieve both conflicting sources.

### Gate E: Answer Gate

- Answer generation cites raw sources.
- Faithfulness checker rejects unsupported claims.
- Observe-only groups still do not receive replies.

## Review Process

Perform at least three reviews before implementation:

1. **Architecture Review**: verifies Go/Python ownership and bounded contexts.
2. **Data/RAG Review**: verifies schemas, retrieval strategy, citations, and
   evaluation metrics.
3. **Operational Safety Review**: verifies observe-only behavior, privacy,
   retry/dead-letter, dashboard access, and loop protection.

Implementation starts only when all P0/P1 findings are resolved or explicitly
accepted as deferred work with owner and deadline.

