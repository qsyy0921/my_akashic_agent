# SPEC-137 Runtime Local Bring-Up Post Migration

## Problem

The Go `agent-runtime` already owns the control-plane APIs that Python expects
at `http://127.0.0.1:8780`, but the migrated workspace currently boots only the
Python process. Local bring-up depends on hidden environment variables and a Go
toolchain path that is not guaranteed to be on `PATH`, so Python falls back to
repeated `HTTP 502` errors even though the runtime code and read-only
diagnostics exist.

## Scope

This slice defines the local bring-up contract for the migrated workspace:

- a repo-owned local launcher for `services/agent-runtime`;
- deterministic runtime env for the current dual-account NapCat setup;
- read-only verification of the key runtime endpoints used by Python and
  cutover preflight;
- explicit handling of missing Telegram bot credentials.

## Non-Goals

- No real QQ or Telegram send cutover.
- No automatic Telegram bot token creation or storage.
- No change to Go/Python ownership boundaries.
- No new mutation executor for outbox, capacity, or priority plans.

## Runtime Launcher Requirements

The repo-owned launcher must:

1. Discover a usable `go.exe` from `PATH` or the standard per-user Go install.
2. Derive the repo root from the script location instead of hardcoding the old
   `E:\agent\akashic` path.
3. Start `go run ./cmd/agent-runtime` from `services/agent-runtime`.
4. Set the minimum local env required for the current workstation:
   - `AKASHIC_RUNTIME_ADDR=127.0.0.1:8780`
   - `AKASHIC_RUNTIME_STATE_DIR=<repo>\.akashic-workspace\agent-runtime`
   - `AKASHIC_SHADOW_AUDIT_PATH=<repo>\.akashic-workspace\shadow\runtime-audit.jsonl`
   - `AKASHIC_BOT_IDS=1049511700,2365524513`
   - `AKASHIC_ONEBOT_WS_URLS=qq=ws://127.0.0.1:3001,qq_1049511700=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002`
   - `AKASHIC_ONEBOT_ACCESS_TOKEN=NcatBot`
   - `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=false`
5. Preserve any pre-existing `TELEGRAM_BOT_TOKEN` or
   `AKASHIC_TELEGRAM_BOT_TOKEN`, but not fabricate one when missing.
6. Fail fast if port `8780` is already occupied by another process.
7. Write stdout/stderr logs under repo `logs/`.

## Verification Requirements

Successful local bring-up must provide evidence for:

- `GET /healthz` returns 200 from `127.0.0.1:8780`.
- `GET /v1/runtime-config` returns 200 and shows OneBot aliases configured.
- `GET /v1/runtime-overview` returns 200.
- `POST /v1/outbound-cutover/readiness` returns 200 and read-only blockers or
  readiness data.
- `POST /v1/outbound-cutover/plan` returns 200 and `side_effect=none`.
- Python-side runtime proxy calls stop failing with `agent runtime HTTP 502`
  after Go runtime is up.

## Telegram Constraint

When no Telegram bot token is configured:

- the runtime may still start successfully;
- diagnostics must report Telegram as not configured or absent;
- review/live-check records must treat Telegram backend smoke as blocked by
  missing credentials rather than as passed.

## Acceptance

- A new operator can launch the runtime from the current repo without manually
  reconstructing the Go path or OneBot env.
- The migrated workspace uses its own `.akashic-workspace` state paths rather
  than the old `E:\agent\akashic` workspace.
- Python regains read-only connectivity to Go runtime control-plane endpoints
  on `127.0.0.1:8780`.
