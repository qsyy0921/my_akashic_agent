# Runtime Overview Delivery Smoke

## Context

Go already exposes `/v1/delivery-smoke/readiness` and outbound cutover gates use
it internally. The main operator entrypoint, `/v1/runtime-overview`, currently
surfaces adapter configuration and cutover plan state, but not the direct smoke
matrix. When QQ/NapCat or Telegram delivery is being cut over, operators should
see the smoke readiness state in the same summary/card shape as other runtime
control-plane diagnostics.

## Boundary

Go owns deterministic delivery infrastructure:

- adapter routing and channel alias resolution;
- read-only smoke readiness matrix;
- runtime overview summary/card/detail fields;
- cutover diagnostics before enabling Go-owned local worker or external lease
  execution.

Python remains responsible for:

- AI reply generation and tool selection;
- deciding whether a message should be sent;
- compatibility workers until explicit cutover;
- provider/model-specific fallback behavior.

The runtime overview aggregation must not send platform messages, lease outbox
deliveries, mutate queue state, or invoke Python AI workers.

## Design

Add an optional `RuntimeOverviewDeps.DeliverySmoke` dependency that uses the
existing `CheckDeliverySmokeReadiness` application port with an empty command.
The delivery smoke service already holds default cases and account channel
aliases from runtime config, so runtime overview does not duplicate config
parsing.

```text
RuntimeOverviewService
  -> DeliverySmoke.CheckDeliverySmokeReadiness(empty command)
  -> RuntimeOverviewView.DeliverySmokeReadiness
  -> Summary:
       delivery_smoke_ready
       delivery_smoke_reason
       delivery_smoke_cases
       delivery_smoke_ready_cases
       delivery_smoke_not_ready_cases
       delivery_smoke_blockers
  -> Card:
       id: delivery_smoke
       value: ready_cases/cases
       status:
         ok when ready
         danger when blockers exist
         warn when cases are present but not all ready
         muted when no cases/defaults exist
```

## HTTP Shape

`GET /v1/runtime-overview` includes:

```json
{
  "summary": {
    "delivery_smoke_ready": false,
    "delivery_smoke_reason": "delivery_smoke_not_ready",
    "delivery_smoke_cases": 2,
    "delivery_smoke_ready_cases": 1,
    "delivery_smoke_not_ready_cases": 1,
    "delivery_smoke_blockers": 1
  },
  "cards": [
    {
      "id": "delivery_smoke",
      "label": "Delivery Smoke",
      "value": "1/2",
      "status": "danger"
    }
  ],
  "delivery_smoke_readiness": {
    "side_effect": "none"
  }
}
```

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Risks

- Empty default smoke cases produce a muted card, not a hard runtime health
  failure. Cutover readiness endpoints still remain the stronger gate.
- Synthetic media smoke only verifies planning and adapter routing; it does not
  prove remote platform upload behavior.
