# Review: Agent Gateway Rename

Spec:

- `docs/sdd/adr/0003-go-service-ddd-granularity.md`
- `docs/sdd/specs/agent-architecture/008-go-migration-scope.md`

Implementation summary:

- Renamed the Go service from `services/message-gateway` to
  `services/agent-gateway`.
- Renamed the command entrypoint from `cmd/message-gateway` to
  `cmd/agent-gateway`.
- Renamed SDD specs from `specs/message-gateway` to `specs/agent-gateway`.
- Updated Go module/import paths and architecture-test module boundary.
- Updated docs to describe Go as the agent infrastructure/control-plane layer,
  not only a visible messaging gateway.

Tests run:

- Pending in this slice before commit: Go unit tests, Go build, Python SDD
  fixture/shadow regression tests, and `git diff --check`.

Findings:

- The old name was accurate for the first slice, but became too narrow after
  outbox, media registry, and generic agent-job control-plane features landed.
- `agent-gateway` preserves the gateway/control-plane boundary without claiming
  ownership of Python-side LLM/tool execution.

Decision:

- Accept the rename now while the service is still small enough to migrate
  cheaply.

Follow-ups:

- Keep Python worker integrations pointed at the `agent-gateway` HTTP API.
- Split future Go services only when a bounded context has independent storage,
  scaling, or release lifecycle.
