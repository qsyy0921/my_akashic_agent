# Goal Verifier Stdout Modes ATDD

## 场景

当 operator 或 Codex 在当前 turn 运行 `scripts/verify-go-migration-goal.ps1`
时，脚本必须优先稳定刷新 `.codex-goal-verifier.json`，而不是要求调用端消费整份
大 JSON stdout。

## 验收步骤

1. 运行：
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
2. 记录 stdout 是否为小摘要 JSON。
3. 运行：
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1 -StdoutMode none`
4. 检查 repo 根目录 `.codex-goal-verifier.json` 的更新时间。
5. 打开 artifact，确认 `current_state`、`migration_residuals`、
   `migration_bucket_summary` 和 `checks.dashboard_read_models` 仍存在。

## 通过条件

- 默认 stdout 只返回小摘要 JSON
- `-StdoutMode none` 仍会刷新 artifact
- artifact schema 不变
- `open_blockers` 与 `current_state` 可直接从当前 turn artifact 读取
