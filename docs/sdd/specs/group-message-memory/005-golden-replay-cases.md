# SPEC-005: Golden Replay Cases

## Status

Draft

## Context

Group-memory and RAG implementation should be validated against replayable
cases before live behavior. These cases should be small enough for tests but
realistic enough to catch failure modes.

## Fixture Directory

Use:

```text
tests/fixtures/group_message_replay/
```

## Required Cases

### `hardware_recommendation_001`

Purpose:

- Verify hardware DIY recommendation extraction and retrieval.

Must include:

- user asks for a budget build;
- several parts are recommended;
- one later correction changes the motherboard or PSU choice;
- expected `ProductConfigMemory`;
- query expects cited answer and time context.

### `game_guide_001`

Purpose:

- Verify game攻略 extraction into `ProcedureMemory`.

Must include:

- scattered steps across multiple users;
- version marker;
- warning that update may invalidate the method;
- expected procedure steps and source thread.

### `image_evidence_001`

Purpose:

- Verify image-only evidence.

Must include:

- image attachment with no useful text message;
- OCR/vision summary;
- expected `AssetInsight`;
- answer must cite asset id.

### `file_attachment_001`

Purpose:

- Verify file summary and source citation.

Must include:

- file metadata;
- extracted document summary;
- answer citing file asset and source message.

### `conflicting_claims_001`

Purpose:

- Verify conflict handling.

Must include:

- two incompatible claims;
- later correction or lack of consensus;
- answer must surface conflict instead of hiding it.

### `version_sensitive_001`

Purpose:

- Verify validity metadata.

Must include:

- old advice;
- new version update;
- supersession relation;
- answer must mention version/time.

### `unresolved_question_001`

Purpose:

- Verify open-question memory.

Must include:

- a question with no answer in the window;
- later unrelated chatter;
- expected `OpenQuestion`;
- retrieval should not hallucinate an answer.

### `observe_only_no_reply_001`

Purpose:

- Verify operational safety.

Must include:

- observed group messages that would normally tempt a reply;
- expected no outbound group message;
- optional owner/private summary only when explicitly queried.

## Metrics Per Case

Each case should define:

- expected memory record types;
- expected source message ids;
- expected source asset ids;
- required answer phrases or fields;
- forbidden unsupported claims;
- retrieval top-k expectations;
- observe-only outbound expectations.

## Acceptance

Implementation can start only when all cases exist as documented JSON fixtures
and are referenced by planned Go/Python tests.

