# Review: agent-job external lease launcher bundle

Spec:
- `docs/sdd/specs/agent-gateway/227-agent-job-external-lease-launcher-bundle.md`

Implementation summary:
- 新增 Go 只读 `launcher-bundle` query/service/handler。
- 新增 repo-owned live smoke，先读取 blocked bundle，再用 bundle 带起 promoted temp runtime。
- unified goal verifier 现在纳入 launcher bundle evidence。

Tests run:
- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_launcher_bundle.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-launcher-bundle.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`

Findings:
- launcher contract 之前仍然分散在 verifier 里，不够 canonical。
- 现在 canonical bundle 已由 Go runtime 只读暴露，live smoke 也改成直接消费 bundle。
- production 剩余 gap 仍是 flags/owner 没切到 live runtime，不是 preflight 或 launcher contract 缺失。

Decision:
- 接受本轮改动。

Follow-ups:
- 仅在明确允许时，对 live runtime 做 production flags/owner preflight。
