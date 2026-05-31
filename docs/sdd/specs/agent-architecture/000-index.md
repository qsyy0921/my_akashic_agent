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
- `008-go-migration-scope.md`: full list of infrastructure responsibilities
  that should move to Go and the retained Python AI runtime scope.
- `009-go-migration-implementation-design.md`: executable phase plan, ports,
  cutover rules, and review checklist for Go migration.
- `010-python-ddd-suitability.md`: why Go uses tactical DDD while Python keeps
  ports/adapters and AI pipeline boundaries instead of full DDD.
- `011-runtime-state-defaults.md`: file-backed default state directory for
  deterministic Go runtime control-plane stores.

Related spec family:

- `../group-message-memory/`: QQ/group message processing, group memory, RAG,
  and evaluation gates.
