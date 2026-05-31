# Phase 8.139 Review: Runtime Overview Delivery Smoke

## Scope

Aggregated Go-owned delivery smoke readiness into `/v1/runtime-overview` so the
main dashboard/operator entrypoint can see QQ/Telegram send-path smoke gates
before outbound cutover.

## Changes

- Added optional `RuntimeOverviewDeps.DeliverySmoke`.
- Added `RuntimeOverviewView.delivery_smoke_readiness`.
- Added summary fields:
  - `delivery_smoke_ready`
  - `delivery_smoke_reason`
  - `delivery_smoke_cases`
  - `delivery_smoke_ready_cases`
  - `delivery_smoke_not_ready_cases`
  - `delivery_smoke_blockers`
- Added `Delivery Smoke` runtime overview card with `ready_cases/cases` value.
- Wired `cmd/agent-runtime` to reuse the existing `deliverySmokeReadiness`
  service and default smoke-case configuration.
- Updated README, SDD index, DONE, LIVE_CHECKS, and BACKLOG.

## Boundary Check

Go owns deterministic delivery readiness:

- channel alias based dispatch planning;
- adapter support checks;
- read-only smoke readiness matrix;
- runtime overview summary/card/detail.

Python remains responsible for:

- generating AI replies;
- deciding whether a message should be sent;
- compatibility worker behavior until explicit cutover.

The change does not send platform messages, lease outbox deliveries, mutate queue
state, or invoke Python AI workers.

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Risks

- Runtime overview uses the default smoke-case matrix. Operators still need the
  dedicated `/v1/delivery-smoke/readiness` endpoint for ad hoc group IDs or
  synthetic media options.
- Smoke readiness proves routing and adapter availability, not remote platform
  delivery success.
