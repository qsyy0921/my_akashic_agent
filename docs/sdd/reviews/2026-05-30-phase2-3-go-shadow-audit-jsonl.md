# Review: Phase 2.3 Go Shadow Audit JSONL

Spec:

- `docs/sdd/specs/agent-architecture/007-shadow-mode-gateway.md`
- `docs/sdd/specs/agent-architecture/005-migration-plan.md`

Implementation summary:

- Add `infrastructure/auditjsonl`, a Go JSONL-backed implementation of the
  audit and shadow query ports.
- Wire `AKASHIC_SHADOW_AUDIT_PATH` in `cmd/message-gateway` so local runs can
  persist shadow audit events across process restarts.
- Include attachment metadata in the shadow observed query so the dashboard can
  render links when the gateway has asset data.
- Keep the default development path in-memory when the environment variable is
  not set.
- Document the new environment variable in the gateway README.

Safety boundaries:

- No QQ/Telegram adapter is moved to Go.
- No production outbox behavior is enabled.
- Python still owns live message bus delivery and agent execution.
- The JSONL store is read/write audit infrastructure only.

Tests run:

- `gofmt -w services\message-gateway\cmd\message-gateway\main.go services\message-gateway\infrastructure\auditjsonl\store.go services\message-gateway\infrastructure\auditjsonl\store_test.go`
  passed.
- `go test ./...` passed under `services/message-gateway`.
- `go build ./cmd/message-gateway` passed under `services/message-gateway`.
- `uv run pytest tests\test_shadow_audit_plugin.py tests\test_shadow_gateway.py tests\test_sdd_contract_fixtures.py -q`
  passed.
- `git diff --check -- services\message-gateway docs\sdd\specs\agent-architecture\007-shadow-mode-gateway.md docs\sdd\reviews\2026-05-30-phase2-3-go-shadow-audit-jsonl.md docs\sdd\reviews\README.md`
  passed.

Decision:

- Proceed as a Phase 2 persistence improvement before any routing cutover.

Follow-ups:

- Add SQLite/Postgres or NATS-backed stores only after the shadow path is stable.
- Add a durable media registry later for download, retention, and access policy.
