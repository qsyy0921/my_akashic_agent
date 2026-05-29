# Review: Go Migration SDD Design

Spec:

- `docs/sdd/specs/agent-architecture/008-go-migration-scope.md`
- `docs/sdd/specs/agent-architecture/009-go-migration-implementation-design.md`
- `docs/sdd/IMPLEMENTATION_FREEZE.md`

Review method:

- Local SDD checklist review.
- Attempted external Claude review failed because the review tool is not logged
  in: `Not logged in - Please run /login`.

Findings:

### P0

None.

### P1

None for the design. The design keeps production platform cutover behind
feature flags, review notes, and runtime verification.

### P2

1. Outbox state is allowed as a control-plane slice, but delivery adapters must
   not be enabled as production senders until shadow comparison and rollback are
   documented.
2. Media registry should be the next design/implementation slice because QQ
   images/files are already visible in the Python compatibility layer and should
   move toward stable Go asset ids.
3. Scheduler migration should stay behind message/media/job boundaries; moving
   it too early would add operational complexity without reducing the current
   platform risk.

Decision:

**Design passes for staged migration.** Start with non-production Go
control-plane slices: outbox state, media registry, job lifecycle, and dashboard
queries. Do not cut over QQ/Telegram production delivery yet.

Follow-ups:

- Add persistent outbox store before adapter cutover.
- Add Go media/file registry SDD spec before changing media behavior.
- Add Python compatibility client tests whenever Python starts calling Go APIs
  for migrated resources.
