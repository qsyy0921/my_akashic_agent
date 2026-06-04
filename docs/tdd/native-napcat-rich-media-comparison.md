# TDD: Native NapCat Rich Media Comparison

## Scope

- Protects the repo-local native comparison workflow used to classify QQ
  rich-media failures.
- Protects the governance conclusion that native parity evidence is required
  before blaming the Go adapter.

## Target Code Paths

- `scripts/run-napcat-native-rich-media-smoke.ps1`
- `services/agent-runtime/infrastructure/onebotdelivery/adapter.go`
- `services/agent-runtime/infrastructure/onebotdelivery/websocket.go`
- `docs/sdd/LIVE_CHECKS.md`
- `docs/sdd/OPEN_ISSUES.md`

## Test Matrix

| Case | Level | Expectation |
| --- | --- | --- |
| native stream upload happy path | manual integration | image/file upload returns `file_path` using the same payload shape as Akashic |
| native rich-media platform failure parity | manual integration | native send returns the same `rich media transfer failed` as Akashic |
| governance index/doc integrity | pytest | new spec/docs remain indexed and governance docs stay parseable |

## Required Automated Tests

- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`

## Deferred Coverage

- Native NapCat smoke itself remains ATDD/live smoke because it depends on real
  QQ accounts, Docker, and live group access.
