# ATDD: Proactive Runtime Flow Live Smoke

## Scope

- 验证 Go-owned proactive deterministic state 在隔离 temp runtime 中可真实写入、
  查询、清理和持久化。

## Preconditions

- 本机可运行 `go run ./cmd/agent-runtime`
- `uv` 可运行 Python verifier
- 不要求 QQ / Telegram token

## Scenarios

### Scenario 1

- Action:
  运行 `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-proactive-runtime-flow-live-smoke.ps1`
- Expect:
  返回 `live_verified`，并包含 deliveries duplicate/count、seen/rejection、
  cleanup、anyaction quota、drift、bg-context、tick-log 的结构化证据。

### Scenario 2

- Action:
  查看 verifier 输出中的 temp `proactive-state.json` 相关计数。
- Expect:
  cleanup 前 deliveries / seen / rejection / context-only 已落盘；
  cleanup 后这些过期记录被删除；
  anyaction / drift / bg-context / tick-log 仍保留。

## Failure Signals

- `conclusion.status != live_verified`
- cleanup 未删除应过期记录
- drift / bg-context / tick-log detail 缺失
- verifier 触发 QQ / Telegram / Python AI 依赖

## Evidence

- verifier JSON
- temp `proactive-state.json`
- temp runtime stdout/stderr logs
