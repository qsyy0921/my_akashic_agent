# TDD: media recovery boundary live verifier

## Scope

本轮不改 media recovery executor 行为，只补 repo-owned live verifier 和 unified goal 接线。

## Test / Verification Targets

1. 脚本级 live verification
   - `.\scripts\verify-media-recovery-boundary.ps1`
   - `.\scripts\verify-go-migration-goal.ps1`

2. 治理文档校验
   - `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Assertions

1. 独立 verifier 必须能读取：
   - runtime overview media recovery summary/detail
   - content diagnostics
   - sample asset access/recovery/preflight
   - dashboard recovery plan proxy
   - dashboard runtime overview

2. unified goal verifier 必须暴露：
   - `media_recovery`
   - `residual_classification.media_recovery_private_source_executor`

3. 当前 live 样本下必须能稳定得出：
   - `file://` 为主
   - `media_asset_content_forbidden`
   - `media_asset_content_recovery_fix_content_roots`
   - `operator_runtime_config`

4. 不新增真实副作用：
   - 不调用 `POST /v1/media-assets/content-recovery`
   - 不创建 approval / mutation
   - 不下载/缓存内容
   - 不触发 OCR/VLM/RAG/AI
