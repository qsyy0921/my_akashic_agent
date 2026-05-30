# Review: Outbox Dispatch Worker Compatibility Slice

Spec:

- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`

Implementation summary:

- Added Python client methods for Go-owned outbox leasing and terminal state
  updates: `/v1/outbox/lease-next`, `/succeeded`, and `/failed`.
- Added `AgentGatewayOutboxWorker` / `AgentRuntimeOutboxWorker` as a compatibility
  dispatcher that leases Go outbox deliveries and sends them through the existing
  Python `message_push` channel registry.
- Added `integrations.agent_runtime.outbox_worker_enabled`, defaulting to
  `false`, so current direct Python sends and QQ observe-only collection are not
  affected until cutover is explicitly enabled.
- Added bootstrap wiring with account-id to channel-name routing for QQ
  multi-account sends.

Tests run:

- Targeted pytest for agent runtime client, outbox worker, and bootstrap wiring.
- Python compile check for changed modules.

Findings:

- This is still a compatibility adapter. Go owns lease, retry visibility, and
  terminal delivery state, but real platform SDK sends still happen in Python.
- The worker intentionally treats `message_push` textual failure results as
  failed delivery state because `message_push` catches platform exceptions and
  returns localized error text.
- The feature is opt-in to avoid duplicate sends while older direct-send paths
  still exist.

Decision:

- Accept this slice as the next incremental step toward Go-owned outbound
  delivery lifecycle.
- Keep `outbox_worker_enabled=false` by default until a reviewed adapter cutover
  removes or gates direct Python sends.

Follow-ups:

- Add shadow-mode comparison for direct Python sends versus Go outbox dispatch.
- Move platform-specific sender adapters behind explicit outbound ports after
  direct-send callers are audited.
- Add dashboard control to show whether the compatibility outbox worker is
  enabled and actively leasing.
