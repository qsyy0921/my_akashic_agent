# Review: agent job strict token mode

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added `WithStrictAgentJobLeaseToken` as an app-service option on
  `AgentJobService`.
- Runtime wires `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true` into the service.
- In strict mode, `running`, `succeeded`, and `failed` reject empty
  `lease_token` values before mutating state.
- Default mode remains compatibility mode, so old workers that do not send
  tokens can still finish state-store leased jobs.
- Runtime logs whether strict token mode is enabled at startup.

Tests run:
- `go test ./...`
- `go test ./app/service -run TestAgentJobServiceStrictLeaseTokenRejectsEmptyResultWriteback -count=1 -v`

Findings:
- Python workers already pass lease tokens from the previous slices, so strict
  mode is now a deploy-time gate rather than a code-level blocker.
- `renew` is already strict because it requires a non-empty token at the domain
  level.

Decision:
Keep strict token mode opt-in until the running deployment is known to use the
updated Python workers. Require it before any future `agent_job` NATS external
lease cutover.

Follow-ups:
- Add exact queue work-id leasing for `agent_job` notifications.
- Add duplicate-delivery smoke before mapping Python result writeback to NATS
  ack/nack/term.
