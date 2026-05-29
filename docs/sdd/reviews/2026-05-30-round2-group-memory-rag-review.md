# SDD Review Round 2: Group Memory And RAG

## Scope

Review the group-message processing design, including but not limited to RAG.

Reviewed:

- `docs/sdd/specs/group-message-memory/001-processing-pipeline.md`
- `docs/sdd/specs/group-message-memory/002-memory-model.md`
- `docs/sdd/specs/group-message-memory/003-rag-architecture.md`
- `docs/sdd/specs/group-message-memory/004-evaluation-and-quality-gates.md`

## Findings

### P0

None.

### P1

1. **RAG is correctly treated as a retrieval layer, not the entire memory
   system.**
   The design includes raw logs, threads, episodic summaries, semantic facts,
   procedural memory, social/policy memory, assets, and retrieval views.

2. **Every answer must be source-cited.**
   This is now an invariant. Implementation should reject uncited group-memory
   answers, especially for hardware recommendations and game攻略.

3. **Evaluation gates are required before live behavior.**
   Synthetic replay, open-source baseline data, and real shadow replay are
   required. This prevents demo-only RAG.

### P2

1. The memory model should later choose concrete storage technologies, but the
   current spec correctly avoids premature database commitment.
2. The trust model is intentionally soft. This is appropriate because group
   reliability is domain-specific and should not become a hardcoded allowlist.
3. The RAG architecture should later include concrete model/provider choices,
   but only after contract fixtures and evaluation cases exist.

## Decision

**Pass for design.**

The group-memory/RAG design is broad enough for QQ groups and specific enough to
write fixtures and evaluation tests. Implementation remains blocked until those
fixtures and gates are created.

## Required Follow-Up Before Code Migration

- Create golden replay cases for:
  - hardware DIY recommendation;
  - game攻略 extraction;
  - image-only evidence;
  - file attachment summary;
  - conflicting advice;
  - outdated version-specific advice;
  - unresolved question.
- Define minimum retrieval and answer metrics.
- Define source citation schema shared by Go and Python.

