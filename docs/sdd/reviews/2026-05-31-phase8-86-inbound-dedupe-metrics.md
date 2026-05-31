# Phase 8.86 Review: Inbound Dedupe Metrics

## Scope

- Added a Go-owned read-only `GET /v1/inbound-dedupe/metrics` endpoint.
- Added `InboundDedupeService.Metrics` to summarize sampled dedupe records by
  active/expired state, duplicate records, total seen count, duplicate
  suppressed seen count, and scope.
- Added inbound dedupe metrics to `GET /v1/runtime-overview` summary and card
  output.
- Added Python `AgentGatewayClient.get_inbound_dedupe_metrics` and dashboard
  normalization for the Go aggregate payload.

## Boundary Check

- This is runtime infrastructure state, so it belongs in Go.
- The metrics path is read-only and reports `side_effect=none`.
- It does not clean expired records, trigger platform sends, run model calls,
  download attachments, or mutate dashboard state.
- Python remains responsible for SDK polling and message conversion; it only
  reads the Go metrics API when needed.

## Tests

- Go service tests cover duplicate suppression metrics.
- Go HTTP tests cover `/v1/inbound-dedupe/metrics`.
- Go runtime overview tests cover summary fields and card status.
- Python client and dashboard tests cover the new runtime payload shape.

## Risks

- Metrics are bounded by the requested sample limit. They are operational
  diagnostics, not long-window analytics.
- Expired records can appear until the next mutating `check` call cleans them;
  this is intentional to keep the metrics endpoint side-effect free.

## Decision

Accept as the runtime control-plane view for inbound dedupe suppression.
