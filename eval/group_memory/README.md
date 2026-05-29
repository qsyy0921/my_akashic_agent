# Group Memory Offline Eval

This folder contains a small offline evaluation entrypoint for the QQ group
memory prototype. The checked-in fixture is intentionally tiny and synthetic so
CI does not depend on network access or large third-party datasets.

The pipeline mirrors the production path:

1. seed observed QQ-group messages into `sessions.db`
2. run `GroupMemoryService.ingest_group`
3. query the structured memory store
4. compare returned topics/statuses with fixture labels

Larger open datasets can be converted into the same JSON shape as
`tests/fixtures/group_memory_open_strategy_dataset.json`.
