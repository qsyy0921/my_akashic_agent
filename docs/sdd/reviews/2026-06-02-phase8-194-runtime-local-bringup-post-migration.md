# Phase 8 Review 194: Runtime Local Bring-Up Post Migration

## Spec

- `docs/sdd/specs/agent-gateway/137-runtime-local-bringup-post-migration.md`

## Scope

- Added a repo-owned local launcher for `services/agent-runtime`.
- Switched runtime launcher/docs to the migrated workspace paths under
  `E:\agent\my-akashic_agent`.
- Fixed Python AI worker status `worker_id` collisions that caused Go runtime
  lease conflicts during local bring-up.
- Verified the read-only runtime and dashboard endpoints needed for local
  preflight.

## Boundary Review

- Go runtime remains the control plane only; no real QQ or Telegram cutover was
  enabled.
- `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED` stays `false`, so Go does not execute
  real outbox sends in this slice.
- Telegram diagnostics were treated as blocked by missing bot credentials; the
  Desktop client login does not count as backend readiness.

## Verification

- `C:\Users\10495\AppData\Local\Programs\Go\bin\go.exe test ./...`
- `uv run pytest tests/test_bootstrap_wiring_p2.py::test_bootstrap_runtime_outbox_worker_is_opt_in tests/test_bootstrap_wiring_p2.py::test_bootstrap_runtime_rag_eval_worker_is_opt_in tests/test_bootstrap_wiring_p2.py::test_bootstrap_runtime_worker_id_suffixes tests/test_sdd_governance_docs.py tests/test_sdd_spec_index.py -q`
- `GET http://127.0.0.1:8780/healthz`
- `GET http://127.0.0.1:8780/v1/runtime-config`
- `GET http://127.0.0.1:8780/v1/runtime-overview?limit=20&event_limit=10&stale_after_seconds=900`
- `POST http://127.0.0.1:8780/v1/outbound-cutover/readiness`
- `POST http://127.0.0.1:8780/v1/outbound-cutover/plan`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview`
- `GET http://127.0.0.1:2236/api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=5`

## Findings

- The migrated workspace can now start Go runtime locally through
  `scripts/start-agent-runtime.ps1` without reconstructing old workspace paths
  or manually locating `go.exe`.
- Python no longer reports fresh `agent runtime HTTP 502` errors once Go
  runtime is up and the main process is restarted against it.
- Local QQ/NapCat preflight is healthy at the adapter level, but outbound
  cutover remains blocked because no real outbox execution path is enabled:
  `outbox_execution_path_not_ready`.
- Telegram backend smoke remains blocked until `TELEGRAM_BOT_TOKEN` is
  configured; current diagnostics correctly report `telegram_token_configured=false`.
- A stale pre-fix worker-status record may remain visible temporarily until its
  lease expires, but the new role-scoped worker IDs are healthy.

## Residual Risk

- This slice proves local bring-up and read-only preflight only; it does not
  prove live QQ send cutover or Telegram bot backend receive/send.
- A broader `tests/test_bootstrap_wiring_p2.py` run still hits pre-existing
  sqlite resource warnings outside the focused assertions used here.
