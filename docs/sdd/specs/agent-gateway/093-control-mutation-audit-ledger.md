# Control Mutation Audit Ledger

## Context

The runtime now has an operator approval ledger and approval check preflight.
The next missing piece before any real control-plane mutation is an immutable
audit trail for mutation attempts and outcomes.

This slice adds a Go-owned control mutation audit ledger. It records planned,
applied, failed or rolled-back mutation records, but it does not execute the
mutation itself. Future cutover, capacity, priority, queue-owner or worker
concurrency mutation endpoints can require both an active approval and a
mutation audit record.

## Ownership Boundary

Go owns:

- deterministic mutation audit records;
- durable mutation audit persistence;
- HTTP API to record/list mutation audits;
- validation of action, target, status, approval id and rollback metadata.

Python owns:

- AI execution and model/provider calls;
- prompt, tool and RAG/Memory pipeline behavior;
- future worker behavior after a separately designed Go-approved mutation.

## Goals

- Add a Go-owned `ControlMutationAudit` domain model.
- Add `POST /v1/control-mutations` and `GET /v1/control-mutations`.
- Persist records in `.akashic-workspace/agent-runtime/control-mutations.json`.
- Keep the endpoint as an audit ledger only.

## Non-Goals

- No configuration mutation.
- No cutover execution.
- No worker startup, concurrency change or autoscaling.
- No queue ack/nack/term.
- No AgentJob mutation.
- No Python AI execution.
- No authorization/identity system.

## API

`POST /v1/control-mutations`

Request fields:

- `target_kind`: stable control-plane target, such as
  `outbound_cutover`, `knowledge_planner_cutover`, `agent_job_capacity`,
  `agent_job_priority`, or `queue_execution_owner`.
- `target_id`: stable target id or plan id.
- `action`: operator-requested action, such as `enable`, `disable`, `rollback`,
  `update_limit`, `switch_owner`, or `recover`.
- `status`: `planned`, `applied`, `failed`, or `rolled_back`.
- `operator_id`: human or automation actor id.
- `approval_id`: required stable approval id.
- `reason`: optional for planned/applied, required for failed/rolled_back.
- `rollback_of`: optional mutation id when this audit record is a rollback.
- `rollback_ref`: optional external rollback reference.
- `metadata`: string map.
- `timestamp`: optional RFC3339 timestamp.

`GET /v1/control-mutations`

Filters:

- `target_kind`
- `target_id`
- `status`
- `approval_id`
- `limit`

## Invariants

- `target_kind`, `target_id`, `action`, `status`, `operator_id`,
  `approval_id`, and `created_at` are required.
- `failed` and `rolled_back` records require a reason.
- `rolled_back` records require `rollback_of` or `rollback_ref`.
- Mutation id is deterministic for timestamp + target + action + status.
- The API has `side_effect=runtime_state_only`.
- The API never mutates runtime configuration or executes worker/queue/AI work.

## Acceptance

- Domain tests validate required fields, status, reason and rollback metadata.
- Service tests record/filter audit records and expose side effect metadata.
- Store tests persist/reload audit records.
- HTTP tests cover POST and GET.
- Full Go tests remain green.
