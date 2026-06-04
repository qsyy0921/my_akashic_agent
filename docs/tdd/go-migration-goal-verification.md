# Go Migration Goal Verification TDD

Date: 2026-06-02

## Scope

为 `scripts/verify-go-migration-goal.ps1` 固化当前 live goal 审计入口。

## Test Targets

1. 脚本能读取当前 live runtime endpoint，而不是只拼静态文案。
2. 脚本会复用：
   - `verify-knowledge-planner-cutover.ps1`
   - `verify-telegram-backend.ps1`
3. 脚本输出统一 JSON，并稳定包含：
   - `qq_outbox_cutover`
   - `qq_rich_media`
   - `telegram_backend`
   - `knowledge_planner`
   - `residual_classification`
   - `checks.open_blockers`
4. `IncludeNativeRichMediaProbe` 关闭时不执行 rich-media probe。
5. `IncludeNativeRichMediaProbe` 打开时，会把原生 NapCat probe 结果并入统一 JSON。

## Verification Commands

```powershell
.\scripts\verify-go-migration-goal.ps1
.\scripts\verify-go-migration-goal.ps1 -IncludeNativeRichMediaProbe -RichMediaProbeGroupId 3219982
uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q
```

## Expected Failure Signals

- 缺字段：脚本输出没有统一 JSON 主体或关键 section 缺失。
- 假阳性：Telegram token 缺失时仍输出 backend ready。
- 假阴性：knowledge planner 明明已运行，但脚本不能复用已有验证入口给出健康结论。
- Rich-media probe 未显式开启却默认发起 probe。
