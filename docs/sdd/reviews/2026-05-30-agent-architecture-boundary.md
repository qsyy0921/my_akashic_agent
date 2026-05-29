# SDD Review: Agent Architecture Boundary

## Scope

Review the proposed split between Go message infrastructure and Python agent
intelligence before further refactoring.

## Reviewed Artifacts

- `docs/sdd/specs/agent-architecture/000-index.md`
- `docs/sdd/specs/agent-architecture/001-current-system-inventory.md`
- `docs/sdd/specs/agent-architecture/002-reference-analysis.md`
- `docs/sdd/specs/agent-architecture/003-python-go-boundary.md`
- `docs/sdd/specs/agent-architecture/004-target-agent-architecture.md`
- `docs/sdd/specs/agent-architecture/005-migration-plan.md`
- `docs/sdd/adr/0002-python-go-agent-boundary.md`

## Findings

### P0

None.

### P1

- Python `infra/channels/qq_channel.py` and `telegram_channel.py` still contain
  platform IO, media handling, outbound send, and agent routing in one layer.
  The migration plan correctly blocks adding new channel responsibilities there.
- Go gateway currently uses in-memory infrastructure. This is acceptable for a
  vertical slice but cannot be the durable source of truth until SQLite/Postgres
  and a queue are introduced.
- Shared JSON contracts do not exist yet. Phase 1 must happen before moving
  adapters to Go.

### P2

- The dashboard remains Python-runtime-centric. It should gradually read
  messages/assets/jobs from Go APIs.
- The Python agent loop has several variants (`passive`, `proactive`, `drift`)
  that share concepts but not enough implementation. Phase 6 should extract
  shared state-machine primitives.
- Tool discovery and skill loading should adopt progressive disclosure more
  strictly to protect prompt size and provider cache behavior.

## Accepted Direction

The proposed split is technically defensible:

- Go is suitable for account routing, loop guard, outbox, queue, audit, media
  registry, scheduler infrastructure, and long-running job state.
- Python is suitable for LLM reasoning, provider routing, prompt/context
  construction, tool semantics, memory/RAG, and skill execution.

## Next Required Specs Before Code Migration

1. `agent-gateway/006-contract-fixtures.md`
2. `agent-runtime/001-turn-state-machine.md`
3. `agent-runtime/002-context-pipeline.md`
4. `agent-runtime/003-tool-catalog-and-policy.md`
5. `agent-gateway/007-media-asset-registry.md`

## Verification

Completed during review:

- Local Akashic module inventory and large-file scan.
- Local Hermes architecture scan.
- Local CCB/Claude Code architecture and loop scan.
- Local CyberClaw/OpenClaw-inspired transparent-agent scan.
- Go gateway tests were already passing after the current vertical slice:
  `go test ./...` and `go build ./cmd/agent-gateway`.
- Targeted Python QQ/message-push tests were already passing after the current
  private-loop-guard fix.

