# Agent Architecture Specs

## Status

Draft

## Scope

This spec family defines the target architecture for Akashic after splitting
messaging infrastructure from agent intelligence.

The design uses:

- Go DDD + hexagonal architecture for durable backend infrastructure.
- Python agent runtime for model reasoning, tool execution, memory, RAG, and
  skill orchestration.
- SDD review gates before moving large behavior across the boundary.

## Specs

- `001-current-system-inventory.md`: current module map and ownership risks.
- `002-reference-analysis.md`: lessons from Hermes, Claude Code/CCB,
  OpenClaw, and CyberClaw.
- `003-python-go-boundary.md`: target responsibilities for Python, Go, and
  frontend.
- `004-target-agent-architecture.md`: proposed runtime architecture and
  contracts.
- `005-migration-plan.md`: incremental refactor plan and acceptance gates.
- `006-contract-fixtures.md`: shared Go/Python fixture requirements before
  implementation.

Related spec family:

- `../group-message-memory/`: QQ/group message processing, group memory, RAG,
  and evaluation gates.
