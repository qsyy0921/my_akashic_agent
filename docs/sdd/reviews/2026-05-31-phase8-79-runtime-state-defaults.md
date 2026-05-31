# Review: runtime state defaults

## Scope

- Changed unconfigured Go `agent-runtime` control-plane stores from memory
  fallback to a shared file-backed default directory:
  `.akashic-workspace/agent-runtime`.
- Covered observe targets, AgentJob, AgentJob events, media assets, send
  ledger, outbox, outbox events, inbox, knowledge checkpoints, and proactive
  state.
- Added `AKASHIC_RUNTIME_STATE_DIR` as the common override and
  `AKASHIC_RUNTIME_STATE_DIR=memory` as the global ephemeral opt-out.
- Kept per-store `AKASHIC_*_DSN` / `AKASHIC_*_PATH` precedence and existing
  `*_DSN=memory` behavior.
- Exposed `AKASHIC_RUNTIME_STATE_DIR` in sanitized runtime config diagnostics.

## Boundaries

- This does not switch queue execution to NATS/RabbitMQ and does not trigger
  platform sends.
- The stores remain local file-backed adapters; Go state remains authoritative
  for local leases and diagnostics.
- Python AI workers still own extraction, RAG upload, VLM/OCR, and model calls.

## Validation

- `go test ./...`
- `go vet ./...`
- `go build ./cmd/agent-runtime`

## Runtime Smoke

- Restart local `agent-runtime` without per-store persistence env.
- Confirm startup logs report the default runtime state directory.
- Confirm `.akashic-workspace/agent-runtime` contains the expected state files
  after runtime initialization and live sync.
- Confirm observe targets synced from Python remain visible after restarting
  only `agent-runtime`.
- Confirm `/v1/runtime-config` includes `AKASHIC_RUNTIME_STATE_DIR` as a
  sanitized environment key.

## Follow-up

- After the operator sends one text, one image, and one file into an
  observe-only QQ group, verify `/v1/observe-capture-diagnostics` remains
  covered after an `agent-runtime` restart.
- Add receiver heartbeat recovery so Go restarts do not leave
  `receiver_connected=false` until Python receiver startup runs again.
- Add retention/compaction policy once inbox/media volumes grow.
