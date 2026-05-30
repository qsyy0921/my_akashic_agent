# Review: Phase 8.75 Runtime Config Diagnostics

## Changes

- Added Go `GET /v1/runtime-config` as a read-only, sanitized runtime
  configuration snapshot.
- The endpoint reports process address/source, bot ids, OneBot/Telegram env
  presence, expected OneBot channel aliases, missing aliases, worker flags, and
  readiness blockers.
- Secret-bearing env values are never returned raw; tokens, DSNs, and URLs are
  redacted.
- Runtime overview now includes a `Runtime Config` card when the Go aggregate is
  available.
- Updated SPEC-014, SPEC-018, and the Chinese TODO.

## Review Notes

- This is deterministic runtime infrastructure and belongs in Go: it reads env,
  normalizes aliases, performs redaction, and exposes side-effect-free readiness
  blockers.
- It does not call OneBot/Telegram APIs and does not send messages.
- The default expected OneBot aliases are `qq` for the primary local account and
  `qq_<bot id>` for additional bot ids; operators can override with
  `AKASHIC_ONEBOT_EXPECTED_CHANNELS`.

## Verification

- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-config-dashboard`

## Decision

Accept as the preflight visibility layer before restarting runtime with the
dual-account OneBot configuration and running live adapter smoke.
