# Phase 8-278 Review: Goal Verifier Stdout Modes

## 本轮变更

- 为 `scripts/verify-go-migration-goal.ps1` 新增 `-StdoutMode summary|full|none`
- 默认 stdout 改为小摘要 JSON
- `-StdoutMode none` 支持 artifact-only 运行
- `.codex-goal-verifier.json` 继续作为完整权威 artifact

## 验证

- `uv run pytest tests/test_verify_go_migration_goal_script.py tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
- `& .\scripts\verify-go-migration-goal.ps1 -StdoutMode none`
- `& .\scripts\verify-go-migration-goal.ps1`

## 结论

当前 unified goal verifier 不再要求交互式调用端消费完整大 JSON 才能拿到本轮结论。
artifact-first 路径已经固定下来，而 `full` 模式仍保留给需要完整 stdout JSON 的场景。
