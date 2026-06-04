# Review: agent-job external lease cutover diff

Spec:
- `docs/sdd/specs/agent-gateway/228-agent-job-external-lease-cutover-diff.md`

Implementation summary:
- 新增 Go 只读 `cutover-diff` query/service/handler。
- 新增 repo-owned live verifier，直接比较 live runtime 与 canonical launcher bundle。
- live verifier 还会在 temp NATS/runtime 中固定 `zero config drift -> worker coverage blocked -> ready` 这条链路。

Tests run:
- `go test ./app/service ./trigger/http ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_cutover_diff.py tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-diff.ps1`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`

Findings:
- 当前 production runtime 的 remaining gap 已可直接读成 machine-readable drift，
  不再需要人工对照 launcher bundle、runtime-config、queue-backend 和 queue-topology。
- 这轮还修正了一个 smoke 断言：temp promoted runtime 必须先造 aged
  `group_memory_extract` pressure，才能稳定证明 worker coverage gate。
- production cutover 仍未开始；当前 live runtime 仍是 blocked state。

Decision:
- 接受本轮改动。

Follow-ups:
- 只有在明确允许时，才按 cutover diff + launcher bundle 对当前 live runtime
  注入 external-lease/result-ack flags 并切 owner。
