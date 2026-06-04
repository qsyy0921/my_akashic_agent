# TDD: NapCat Session Refresh Rich Media Recheck

## Scope

- Protects the governance around using a container restart as the minimal
  session-refresh attempt before requiring manual QQ re-login.

## Target Code Paths

- `scripts/run-napcat-native-rich-media-smoke.ps1`
- `docs/sdd/DONE.md`
- `docs/sdd/OPEN_ISSUES.md`
- `docs/sdd/LIVE_CHECKS.md`
- `docs/sdd/REMAINING_GO_MIGRATION.md`
- `docs/sdd/TODO.md`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| receiver reconnect after restart | manual integration | QQ receiver returns to connected after `docker restart napcat` |
| rich-media parity after restart | manual integration | native image/file send still reports identical platform failure if restart is insufficient |
| governance/index integrity | pytest | new spec and review entries remain indexed and docs stay parseable |

## Required Automated Tests

- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Real manual QQ re-login remains live smoke outside repo automation.
