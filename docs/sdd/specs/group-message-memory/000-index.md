# Group Message Memory Specs

## Status

Draft

## Scope

This spec family defines how Akashic should process QQ group messages into
useful long-term group memory, including but not limited to RAG.

The design targets:

- observe-only QQ groups;
- hardware DIY groups;
- game/community groups;
- image and file-heavy discussions;
- future Telegram/Feishu/WeChat group sources.

## Specs

- `001-processing-pipeline.md`: end-to-end group message processing pipeline.
- `002-memory-model.md`: layered group memory model and schemas.
- `003-rag-architecture.md`: retrieval, generation, citation, and evaluation
  architecture.
- `004-evaluation-and-quality-gates.md`: test datasets, metrics, and release
  gates before implementation.
- `005-golden-replay-cases.md`: required replay scenarios before implementation.
