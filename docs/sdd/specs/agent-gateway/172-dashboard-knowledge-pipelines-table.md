# 172 Dashboard Knowledge Pipelines Table

## Context

`/v1/runtime-overview` 已透传 Go-owned `knowledge_pipelines` detail，但 runtime overview dashboard panel 仍主要依赖 raw JSON，不利于直接检查 observe-only 群的 capture warning、knowledge freshness、RAG ingest 空转和 worker coverage。

## Decision

Python dashboard panel 为 `knowledge_pipelines` card 增加结构化只读 drilldown，直接展示：

- `targets/enabled/ready/warning/blocked/lagging/stale_checkpoints/receiver_connected` totals
- 每个 pipeline 的 `target_id/account/capture_status/group_memory/rag_ingest/coverage/latest_source_seq/reasons`
- notes

## Constraints

- 只读展示，不新增任何 mutation UI
- 不创建、修改或删除 checkpoint
- 不创建、租约或执行 `group_memory_extract` / `rag_ingest`
- 不触发 QQ/Telegram 发送
- 不触发 Python AI、OCR/VLM、RAG parse 或 enrichment

## Verification

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-go-migration-goal.ps1`
