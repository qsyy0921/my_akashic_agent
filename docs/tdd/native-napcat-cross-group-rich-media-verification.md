# TDD: Native NapCat Cross-Group Rich Media Verification

## Scope

- Protects the documentation and governance around cross-group QQ rich-media
  parity verification.
- Protects the conclusion that repeated native failures across groups narrow the
  blocker to the current NapCat/QQ session rather than Akashic adapter logic.

## Target Code Paths

- `scripts/run-napcat-native-rich-media-smoke.ps1`
- `docs/sdd/TODO.md`
- `docs/sdd/DONE.md`
- `docs/sdd/LIVE_CHECKS.md`
- `docs/sdd/OPEN_ISSUES.md`
- `docs/sdd/REMAINING_GO_MIGRATION.md`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| cross-group native parity repeatability | manual integration | multiple groups reproduce identical post-staging `rich media transfer failed` results |
| text-only owner remains narrow | manual integration | Go outbox owner remains safe for text while rich media stays blocked |
| governance/index integrity | pytest | new spec and review entries remain indexed and governance docs stay parseable |

## Required Automated Tests

- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Actual NapCat re-login, account rotation, and fresh-session validation remain
  live smoke because they depend on real QQ session state outside the repo.
