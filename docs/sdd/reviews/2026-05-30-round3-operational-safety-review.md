# SDD Review Round 3: Operational Safety

## Scope

Review operational safety before any implementation work: observe-only groups,
bot loop prevention, asset privacy, retries, dead letters, and dashboard access.

Reviewed:

- `docs/sdd/specs/agent-architecture/003-python-go-boundary.md`
- `docs/sdd/specs/agent-architecture/004-target-agent-architecture.md`
- `docs/sdd/specs/agent-architecture/005-migration-plan.md`
- `docs/sdd/specs/group-message-memory/*`
- `docs/sdd/adr/0002-python-go-agent-boundary.md`

## Findings

### P0

None.

### P1

1. **Observe-only must be a hard invariant.**
   Group-memory extraction, RAG indexing, OCR, and vision jobs must never emit
   group replies. The only allowed outbound target during observe-only phase is
   an owner/private channel after an explicit owner query.

2. **Assets need controlled access.**
   QQ images/files must be registered with stable asset ids, but dashboard links
   cannot be public unauthenticated file paths. Use local authenticated routes or
   signed temporary URLs.

3. **LLM processing must include redaction.**
   Before Python sends group content to model providers, obvious secrets, phone
   numbers, tokens, and sensitive file content should pass through a redaction
   stage.

4. **Long-running jobs must be recoverable.**
   Image generation, RAG ingestion, asset OCR, and memory extraction must have
   pending/running/succeeded/failed/dead-letter states before they are used for
   production behavior.

### P2

1. Audit trails should include policy decisions, not just raw events.
2. Dashboard should show job errors and source citations so failures are
   inspectable.
3. If multiple QQ accounts are active, account id must be part of every route,
   job, source, and asset id.

## Decision

**Conditional pass.**

The design covers the safety risks, but code implementation is not authorized
until the implementation-freeze checklist is written and contract fixtures exist.

## Required Follow-Up Before Code Migration

- Add a short SDD implementation freeze checklist.
- Add contract fixture spec and tests.
- Add explicit observe-only test cases to golden group replays.
- Add asset URL/auth policy to the media registry spec.

