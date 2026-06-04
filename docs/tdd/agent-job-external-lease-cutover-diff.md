# TDD: agent-job external lease cutover diff

## Tests

- Go service test:
  - blocked runtime drift
  - fully matched promoted runtime ready
- HTTP handler test:
  - `/v1/agent-job-external-lease/cutover-diff` returns structured drift payload
- Python verifier test:
  - successful `live_verified` aggregation
  - failed aggregation when any check is false
- unified goal verifier test:
  - `.codex-goal-verifier.json` wiring for
    `agent_job_external_lease_cutover_diff`
