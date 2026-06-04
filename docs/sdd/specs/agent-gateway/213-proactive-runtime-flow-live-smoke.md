# 213 Proactive Runtime Flow Live Smoke

最后更新：2026-06-03

## 问题

当前 Go runtime 已暴露 proactive state 的只读 endpoint，并且 unified goal
verifier 已能读取 tick logs、quota、drift summary、bg-context 和
context-only 的当前快照。

但这还不等于：

1. Go-owned proactive deterministic state 的写入路径在当前代码库里有
   repo-owned live smoke；
2. seen/rejection/cleanup、anyaction quota、drift/bg-context、tick-log
   这些状态流转已经在隔离 runtime 中被真实串起来验证；
3. unified goal verifier 可以引用比“当前线上快照可读”更强的 current-turn
   证据。

## 目标

新增一个 repo-owned isolated live verifier：

- 启动隔离 temp `agent-runtime`
- 只操作 Go-owned proactive deterministic state
- 不触发 QQ/Telegram 发送
- 不触发 Python AI / proactive loop / LLM 推理

并验证以下流转：

1. deliveries duplicate/count
2. seen-items normalized hit
3. rejection cooldown
4. cleanup 对 deliveries / seen / context-only / rejection 的过期清理
5. anyaction quota snapshot + increment
6. drift mark / finish / summary / skill-state
7. bg-context global mark
8. tick-log start / step / finish / list / detail / steps

## 非目标

- 不验证 Python proactive loop 的语义候选、概率策略、LLM 决策或真正主动发送。
- 不新增 autoscaling / concurrency / priority executor。
- 不修改 dashboard UI。

## 设计

### 入口

- Python verifier:
  `scripts/verify_proactive_runtime_flow_live_smoke.py`
- PowerShell wrapper:
  `scripts/verify-proactive-runtime-flow-live-smoke.ps1`

### 运行方式

1. verifier 拉起隔离 temp `agent-runtime`
2. temp runtime 只使用独立 `AKASHIC_RUNTIME_STATE_DIR`
3. 清除 QQ / Telegram / outbox / knowledge planner 相关 env
4. 仅通过 HTTP 调用 `/v1/proactive/*`

### 证据

verifier 输出结构化 JSON，至少包含：

- `delivery_seen_cleanup_smoke`
- `anyaction_drift_bg_context_smoke`
- `tick_log_smoke`
- `checks`
- `conclusion`
- `state_file/stdout_log/stderr_log`

### 不变式

1. 所有 mutation 都只写 temp runtime state。
2. 不要求也不允许连接 QQ / Telegram adapter。
3. 不启动 Python proactive loop。
4. 不触发 LLM/provider/tool/OCR/VLM/image generation。
5. state file 必须能反映 proactive deterministic state 的持久化。

### Unified verifier 接入

`scripts/verify-go-migration-goal.ps1` 必须把该 verifier 的 current-turn
结果并入：

- 顶层 `proactive_runtime_flow_smoke`
- `residual_classification.proactive_flow_smoke`

但不改变现有 `migration_residuals` 的四桶范围；这条 smoke 是对当前
`proactive` live evidence 的增强，而不是新增迁移 residual 类别。

## 验收

1. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-proactive-runtime-flow-live-smoke.ps1`
   返回 `conclusion.status=live_verified`
2. 该结果证明 deliveries / seen / rejection / cleanup / anyaction / drift /
   bg-context / tick-log 全部通过
3. `scripts/verify-go-migration-goal.ps1` 能输出
   `proactive_runtime_flow_smoke` 和
   `residual_classification.proactive_flow_smoke`
