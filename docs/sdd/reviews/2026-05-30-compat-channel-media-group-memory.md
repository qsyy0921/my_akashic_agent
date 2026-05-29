# Review: Compatibility Channel, Media, and Group Memory Bundle

Spec:

- `docs/sdd/specs/agent-architecture/004-target-agent-architecture.md`
- `docs/sdd/specs/agent-architecture/005-migration-plan.md`
- `docs/sdd/specs/group-message-memory/001-processing-pipeline.md`
- `docs/sdd/specs/group-message-memory/004-evaluation-and-quality-gates.md`

Implementation summary:

- Consolidate the existing Python compatibility work for named QQ accounts,
  peer-bot loop protection, current-channel reply routing, and resilient channel
  startup.
- Add outbound-only Feishu and WeChat webhook channels.
- Capture QQ image/file metadata and expose workspace upload attachments through
  the dashboard.
- Add ChatGPT image proxy tooling and RAGFlow client/tool integration behind
  explicit configuration.
- Add a first local group-memory service, replay/eval fixtures, and tools for
  observe-only QQ group knowledge extraction.

Safety boundaries:

- This is still a compatibility layer, not the target Go routing cutover.
- Observe-only group behavior remains a no-reply invariant.
- Feishu and WeChat channels are outbound webhook senders only.
- ChatGPT proxy and RAGFlow tools are disabled unless configured.
- The Python QQ channel remains overloaded and should be reduced after Go
  gateway/media registry migration.

Tests run:

- `uv run pytest tests\test_channel_clients.py tests\test_bootstrap_wiring_p2.py tests\test_dashboard_api.py tests\test_runtime_smoke.py tests\test_support_modules.py tests\test_chatgpt_proxy_tool.py tests\test_group_memory.py tests\test_ragflow_integration.py -q`
  passed after clearing test pollution from `TELEGRAM_BOT_TOKEN`.

Decision:

- Commit as a tested compatibility bundle so the repository matches the local
  running project, then continue migrating responsibilities into Go in smaller
  slices.

Follow-ups:

- Move QQ media/file registry to Go.
- Move outbox delivery and retry to Go.
- Replace rule-based group-memory extraction with jobized extractor/RAG workers
  after replay quality gates are stricter.
