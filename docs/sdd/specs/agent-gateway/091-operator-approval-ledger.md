# Operator Approval Ledger

## Context

Several future runtime control-plane changes require explicit operator
acknowledgement before they can become real mutations:

- AgentJob result-ack cutover;
- outbound delivery cutover;
- knowledge planner cutover;
- future priority/concurrency/autoscaling changes.

The project currently has read-only readiness and plan endpoints, but no stable
Go-owned place to record that a human operator reviewed and approved a plan.
Without this, future control-plane automation would either rely on Python-local
state or mutate configuration without an auditable acknowledgement.

## Ownership Boundary

Go owns:

- deterministic operator approval records;
- durable approval ledger persistence;
- HTTP API to record/list approvals;
- validation of approval target, decision, actor and expiry.

Python owns:

- AI worker execution;
- model calls, prompts, Memory/RAG/OCR/VLM/image-generation pipelines;
- dashboard display or optional client usage of the ledger.

## Goals

- Add a Go-owned `OperatorApproval` domain model.
- Add `POST /v1/operator-approvals` and `GET /v1/operator-approvals`.
- Persist approvals in the runtime state directory.
- Keep the endpoint as an audit ledger only.

## Non-Goals

- No configuration mutation.
- No cutover execution.
- No worker startup, autoscaling or concurrency changes.
- No queue ack/nack/term, AgentJob mutation, outbox mutation, or platform send.
- No Python AI execution.

## API

`POST /v1/operator-approvals`

Request fields:

- `target_kind`: stable control-plane target, such as
  `agent_job_external_lease_plan`, `outbound_cutover_plan`,
  `knowledge_job_planner_cutover_plan`, `agent_job_capacity_plan` or
  `agent_job_priority_plan`.
- `target_id`: caller-defined stable target id, usually a decision/check hash or
  plan id.
- `decision`: `approved`, `rejected`, or `revoked`.
- `operator_id`: human or automation actor id.
- `reason`: required for rejected/revoked, optional for approved.
- `expires_at`: optional RFC3339 timestamp.
- `metadata`: string map.
- `timestamp`: optional RFC3339 timestamp.

`GET /v1/operator-approvals`

Filters:

- `target_kind`
- `target_id`
- `decision`
- `limit`

## Invariants

- Approval id is deterministic for the event timestamp and target, not reused as
  mutable target state.
- `target_kind`, `target_id`, `decision`, `operator_id`, and `created_at` are
  required.
- Rejection and revocation require a reason.
- Expired approvals remain in the ledger but are returned with `active=false`.
- The API has `side_effect=runtime_state_only`.

## Acceptance

- Domain tests validate required fields, decisions, expiry and sorting.
- Service tests record and list approvals.
- Store tests persist/reload approval records.
- HTTP tests cover POST and GET.
- Full Go tests remain green.
