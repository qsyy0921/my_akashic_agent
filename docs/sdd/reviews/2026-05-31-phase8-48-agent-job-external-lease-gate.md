# Review: agent job external lease gate

Spec:
`docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Extended `/v1/queue-backend` external lease diagnostics with
  `execution_scope`, `allowed_work_kinds`, and `blocked_work_kinds`.
- Kept `AllowExecution=true` only for the current outbox executor after all
  explicit gates pass, and made the scope visible as `outbox_delivery_only`.
- Added `agent_job` as an explicit blocked work kind with the required protocol:
  exact job-id lease, lease token, worker heartbeat, idempotent result writeback,
  and ack-after-result mapping.
- Kept `AgentJobQueueSource=agent_job_state_store` even when outbox external
  lease is ready.

Tests run:
- `gofmt -w services\agent-runtime\app\query\queue_backend.go services\agent-runtime\cmd\agent-runtime\queue_backend.go services\agent-runtime\cmd\agent-runtime\main_test.go`
- `go test ./...`
- `go build ./cmd/agent-runtime`

Findings:
- Outbox can safely map Go dispatch terminal state to NATS `ack` / delayed
  `nack` / `term`.
- Generic `agent_job` cannot use the same executor boundary because Python owns
  the long-running side effect and final result writeback.
- A queue ack at Go lease time would risk lost work; a queue ack before Python
  idempotency is proven would risk duplicated AI side effects.

Decision:
Do not move `agent_job` to NATS external lease in this slice. Keep state-store
lease as the authoritative job discovery path, and expose the blocked state in
runtime diagnostics so operators cannot mistake outbox cutover for generic job
cutover.

Follow-ups:
- Design an `agent_job` result-ack protocol with lease token, heartbeat, and
  duplicate-delivery contract smoke before implementing external queue
  execution for Python workers.
