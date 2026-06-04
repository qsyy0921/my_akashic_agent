# 221. Goal Verifier Stdout Modes

## 背景

`scripts/verify-go-migration-goal.ps1` 现在是当前仓库统一的 current-turn
live evidence 入口，并会默认回写 repo 根目录 `.codex-goal-verifier.json`。

但脚本在 stdout 上直接输出完整 JSON artifact 时，输出体量已经足够大，会让
Codex/PowerShell 调用端把“artifact 已写好”和“stdout 仍在持续刷大 JSON”混在一起。
这会让本轮验证路径脆弱：

- artifact 其实已经刷新；
- 但交互式调用仍可能因为等待完整 stdout 而显得超时或卡住。

## 目标

为 unified goal verifier 增加显式 stdout mode，使 repo-owned live evidence
保留为 `.codex-goal-verifier.json`，同时把交互式调用从“大 JSON stdout 依赖”切换成
更稳定的 artifact-first 路径。

## 非目标

本次不做以下事情：

- 不改变 `.codex-goal-verifier.json` 的 schema
- 不移除任何现有 `current_state` / `migration_residuals` / `dashboard_read_models` 字段
- 不改动任何子 verifier 的 live logic
- 不改变 Go runtime、dashboard read-model、QQ/Telegram/AgentJob cutover 状态

## 设计

### 1. 新增显式 stdout mode

`scripts/verify-go-migration-goal.ps1` 新增参数：

- `-StdoutMode summary`：默认。stdout 只输出一个小摘要 JSON。
- `-StdoutMode full`：保留旧行为，stdout 输出完整 artifact JSON。
- `-StdoutMode none`：只写 `.codex-goal-verifier.json`，stdout 不输出 artifact 内容。

### 2. artifact 继续是权威输出

无论 stdout mode 如何，脚本都必须：

1. 先完整构建 `$result`
2. 把完整 artifact 写入 `JsonOutputPath`
3. 再按 stdout mode 决定是否输出 `summary/full/none`

### 3. 默认 summary 必须足够小但仍可用于当前 turn 判断

默认 `summary` stdout 至少包含：

- `status`
- `artifact_path`
- `generated_at`
- `goal_ready_to_close`
- `open_blockers`
- `current_state`
- `migration_bucket_summary`

这样交互式调用无需再消费完整大 JSON，也不会丢失本轮最关键结论。

## 验收

满足以下条件即视为完成：

1. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
   默认输出小摘要，而不是完整大 JSON。
2. `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode none`
   仍会刷新 `.codex-goal-verifier.json`。
3. `.codex-goal-verifier.json` 中既有 `current_state`、`migration_residuals`、
   `migration_bucket_summary`，也保留 `dashboard_read_models` 与
   `checks.dashboard_read_models`。
4. focused tests 覆盖 stdout mode wiring 与 artifact-first 语义。
