# ATDD: Native NapCat Rich Media Comparison

## Scope

- Proves whether QQ rich-media failure belongs to native NapCat/QQ session
  capability or to Akashic Go adapter logic.
- Confirms whether current evidence is strong enough to allow default Go
  execution-owner cutover.

## Preconditions

- Docker Desktop running locally.
- NapCat container for QQ `1049511700` reachable on `ws://127.0.0.1:3001`.
- Access token available for the WebSocket endpoint.
- Real QQ group available for smoke, currently `27234224`.
- `services/agent-runtime` already configured for the same QQ account.

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\run-napcat-native-rich-media-smoke.ps1`.
- Expect:
  Script returns raw JSON including `image_upload`, `image_send`,
  `file_upload`, and `group_file`.

### Scenario 2

- Action:
  Compare native OneBot `image_send` / `group_file` failure with the latest
  Akashic `run-qq-group-live-smoke.ps1` rich-media failure.
- Expect:
  If both fail with the same `rich media transfer failed`, classify the blocker
  as NapCat/QQ session/platform capability.

### Scenario 3

- Action:
  Read `/v1/outbound-cutover/readiness` and queue/execution-owner diagnostics
  after the native comparison.
- Expect:
  If rich media still fails natively and the runtime only supports a global
  local outbox worker toggle, default Go cutover remains blocked.

## Failure Signals

- Native script cannot reproduce the staged `upload_file_stream` path.
- Native script succeeds but Akashic still fails on the same group and asset
  types.
- Native script fails for reasons unrelated to rich media send, leaving the
  comparison inconclusive.

## Evidence

- Script output from `scripts/run-napcat-native-rich-media-smoke.ps1`.
- Akashic QQ group smoke output from `scripts/run-qq-group-live-smoke.ps1`.
- `/v1/outbound-cutover/readiness`
- `/v1/queue-backend`
