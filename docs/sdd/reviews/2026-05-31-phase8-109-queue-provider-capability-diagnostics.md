# Phase 8.109 Review: Queue Provider Capability Diagnostics

## Spec

- `docs/sdd/specs/agent-gateway/052-queue-provider-capability-diagnostics.md`
- `docs/sdd/TODO.md`

## Implementation Summary

- Added `QueueProviderCapabilityView` to the Go queue backend query contract.
- Extended `/v1/queue-backend` composition to expose:
  - full provider capability matrix;
  - selected provider capability;
  - NATS JetStream as current recommended first external MQ;
  - local state-store as the default safe mode;
  - Redis Streams and RabbitMQ as planned provider-neutral adapter boundaries.
- Captured support flags for shadow publish, dual-read compare, external lease,
  agent_job result-ack, concurrent consumers, and delayed nack.
- Added command-level tests covering default local diagnostics and NATS provider
  capability reporting.

## Tests Run

- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*Provider|TestQueueBackendViewFromEnvDefaultsLocal|TestQueueBackendViewFromEnvNormalizesNATSJetStream" -count=1 -v`
- `go test ./...`

## Findings

- No code-review blockers found.
- The change is read-only diagnostics and does not alter queue cutover gates,
  NATS consumer behavior, Python workers, or platform sending.
- Provider-specific details remain in command/runtime composition and query DTOs;
  domain model and app ports stay provider-neutral.

## Decision

Approved. This makes the MQ decision visible through the Go runtime API instead
of leaving it only in comments or environment names, while keeping NATS as the
first implemented external queue and Redis/RabbitMQ as future adapters.

## Follow-ups

- Only implement Redis Streams or RabbitMQ adapters when a concrete deployment
  requirement appears.
- Continue using explicit smoke gates before expanding `external_lease` cutover.
