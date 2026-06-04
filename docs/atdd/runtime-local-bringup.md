# ATDD: runtime local bring-up

## Scope

- Local operator bring-up of `services/agent-runtime` inside
  `E:\agent\my-akashic_agent`.
- Read-only verification that Python can reach the Go control plane again.

## Preconditions

- Docker Desktop is running.
- NapCat dual-account containers are logged in and expose `3001` / `3002`.
- Python `main.py` process is already running on `127.0.0.1:8765` and dashboard
  `2236`.
- No other process occupies `127.0.0.1:8780`.
- `TELEGRAM_BOT_TOKEN` may be missing; this is an allowed blocked case.

## Scenarios

### Scenario 1

- Action:
  Run the repo-owned PowerShell launcher for `agent-runtime`.
- Expect:
  The launcher discovers `go.exe`, starts `services/agent-runtime`, writes logs
  under `logs/`, and `GET /healthz` on `127.0.0.1:8780` returns 200.

### Scenario 2

- Action:
  Query `GET /v1/runtime-config` and `GET /v1/runtime-overview`.
- Expect:
  Both return 200, OneBot aliases include the current QQ setup, and Telegram is
  reported as not configured when token is missing rather than as healthy.

### Scenario 3

- Action:
  Query `POST /v1/outbound-cutover/readiness` and `POST /v1/outbound-cutover/plan`.
- Expect:
  Both return 200, stay read-only, and expose current blockers or next steps
  without sending QQ/Telegram messages.

### Scenario 4

- Action:
  Observe the running Python process after Go runtime is up, then query a
  Python-facing runtime proxy endpoint.
- Expect:
  New `agent runtime HTTP 502` errors stop appearing in the Python log, and the
  dashboard/runtime proxy can reach Go successfully.

## Failure Signals

- Launcher cannot find `go.exe`.
- Port `8780` stays closed after launch.
- Runtime endpoints return `502`, `503`, or connection refused.
- Python continues to emit fresh `agent runtime HTTP 502` errors after the Go
  runtime is healthy.
- Telegram is reported as healthy without a token.

## Evidence

- Launcher stdout/stderr log paths.
- `Get-NetTCPConnection` or HTTP responses proving `8780` is listening.
- Captured JSON from runtime-config, runtime-overview, readiness, and plan.
- Python log tail after runtime startup.
