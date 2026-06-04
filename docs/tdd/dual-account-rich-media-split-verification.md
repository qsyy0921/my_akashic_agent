# TDD: Dual-Account Rich Media Split Verification

## Scope

- Protects the governance and evidence chain for splitting the QQ rich-media
  blocker by account and media kind.

## Target Code Paths

- `scripts/run-napcat-native-rich-media-smoke.ps1`
- `services/agent-runtime/infrastructure/onebotdelivery/adapter.go`
- `docs/sdd/DONE.md`
- `docs/sdd/OPEN_ISSUES.md`
- `docs/sdd/LIVE_CHECKS.md`
- `docs/sdd/REMAINING_GO_MIGRATION.md`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| second-account native group discovery | manual integration | second account exposes at least one QQ group |
| second-account file parity | manual integration | native and Akashic file send both succeed on second account |
| second-account image parity | manual integration | native and Akashic image send both fail with the same platform error |
| governance/index integrity | pytest | new spec and review entries remain indexed and docs stay parseable |

## Required Automated Tests

- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Per-account execution-owner gating remains deferred because the current
  runtime exposes only global kind gates.
