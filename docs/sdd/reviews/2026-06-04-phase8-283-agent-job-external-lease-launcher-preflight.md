# Review: agent-job external lease launcher preflight

Spec:
- `docs/sdd/specs/agent-gateway/226-agent-job-external-lease-launcher-preflight.md`

Implementation summary:
- 为 `scripts/start-agent-runtime.ps1` 增加 external-lease / result-ack flags 与自定义 runtime addr/state/shadow 参数。
- `/v1/runtime-config` 现在暴露 cutover/smoke booleans 的 presence/redacted value。
- 新增 repo-owned launcher live smoke，使用 `start-agent-runtime.ps1 -Foreground` 起隔离 temp runtime，验证 queue/result-ack flags 注入与 approval-bound preflight。
- unified goal verifier 现在把 launcher preflight 证据并入 current-turn artifact。

Tests run:
- `go test ./cmd/agent-runtime`
- `uv run pytest tests/test_verify_agent_job_external_lease_launcher_preflight.py tests/test_verify_go_migration_goal_script.py -q`
- `uv run python -c "...run_agent_job_external_lease_launcher_preflight(...)"` -> `live_verified`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-launcher-preflight.ps1 | Set-Content ...`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode summary`

Findings:
- launcher 注入层之前确实缺少 result-ack preflight 所需 flags。
- live smoke 更稳的做法是通过 launcher 的 `-Foreground` 路径由 verifier 自己控进程与健康检查；这条 smoke 的目标是验证参数注入，不是复刻 launcher 的后台控制台行为。
- 当前 production gap 没变化，仍然是 flags / owner 尚未切到 live runtime。

Decision:
- 接受本轮改动。

Follow-ups:
- 只有在明确允许时，再把这些 flags 注入当前 live runtime 做 production preflight。
