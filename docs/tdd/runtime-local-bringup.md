# TDD: runtime local bring-up

## Scope

- Protect the current local bring-up contract for `agent-runtime` and the
  existing read-only runtime diagnostics it exposes to Python.

## Target Code Paths

- `services/agent-runtime/cmd/agent-runtime/main.go`
- `services/agent-runtime/cmd/agent-runtime/runtime_config.go`
- `services/agent-runtime/cmd/agent-runtime/delivery_adapters.go`
- `services/agent-runtime/trigger/http/handler.go`
- `scripts/start-agent-runtime.ps1`
- `services/agent-runtime/README.md`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| launcher-compatible runtime env | unit | existing Go env parsing keeps OneBot aliases, state dir, and token redaction stable |
| read-only runtime endpoints | integration | runtime-config, runtime-overview, outbound-cutover readiness/plan stay reachable and side-effect free |
| missing Telegram token | unit/integration | diagnostics mark Telegram missing/unconfigured instead of pretending health |
| migrated repo paths | doc/regression | local runbook and launcher use current repo-derived paths instead of the old workspace root |

## Required Automated Tests

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...`
- `uv run pytest tests/test_sdd_governance_docs.py tests/test_sdd_spec_index.py -q`

## Deferred Coverage

- Actual QQ send smoke remains ATDD/live smoke only; it must not be treated as
  automated coverage.
- Telegram backend send verification remains blocked until a real
  `TELEGRAM_BOT_TOKEN` is provided.
