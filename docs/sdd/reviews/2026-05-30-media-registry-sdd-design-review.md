# Review: Media Registry SDD Design

Spec:

- `docs/sdd/specs/agent-gateway/007-media-file-registry.md`
- `docs/sdd/specs/agent-architecture/009-go-migration-implementation-design.md`

Findings:

### P0

None.

### P1

None for a metadata-only first slice. The spec explicitly blocks arbitrary local
file exposure and keeps content serving behind a later access-policy review.

### P2

1. The first implementation should register metadata only and avoid downloading
   bytes synchronously during inbound message handling.
2. The asset id format includes source message id and index; code should also
   accept caller-provided asset ids for fixtures and existing Python-captured
   attachments.
3. Persistence should follow soon after the in-memory service because dashboard
   asset links are less useful if lost on restart.

Decision:

**Design passes for a metadata-only Go registry slice.**

Follow-ups:

- Implement Go `MediaAsset` domain/app/API with in-memory store.
- Add content route only after workspace path and signed URL policy are tested.
- Add Python shadow mirror from QQ attachments to the registry.
