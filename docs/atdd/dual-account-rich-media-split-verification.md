# ATDD: Dual-Account Rich Media Split Verification

## Scope

- Distinguishes account-specific rich-media problems from broader platform
  failures by comparing account `1049511700` and `2365524513`.
- Confirms whether Akashic and native NapCat agree on second-account behavior.

## Preconditions

- Both NapCat containers are reachable:
  - `ws://127.0.0.1:3001`
  - `ws://127.0.0.1:3002`
- Second account `2365524513` has at least one accessible QQ group.
- Go runtime remains reachable on `127.0.0.1:8780`.

## Scenarios

### Scenario 1

- Action:
  Query second-account group list from `ws://127.0.0.1:3002`.
- Expect:
  At least one group is available for verification.

### Scenario 2

- Action:
  Run native NapCat rich-media smoke on second account, for example:
  `.\scripts\run-napcat-native-rich-media-smoke.ps1 -WebSocketUrl ws://127.0.0.1:3002 -GroupId 284331268`
- Expect:
  File upload and send can succeed, while image send still fails with
  `rich media transfer failed`.

### Scenario 3

- Action:
  Run Akashic manual delivery-dispatch on the same second-account group for one
  file case and one image case.
- Expect:
  File case succeeds through Akashic as well; image case fails with the same
  platform error as native NapCat.

## Failure Signals

- Second account has no group available for verification.
- Native second-account file send fails in the same way as first account file
  send, leaving account split unresolved.
- Akashic and native NapCat disagree on second-account file behavior.

## Evidence

- Second-account group list output
- Native second-account rich-media smoke output
- Akashic manual file/image dispatch output
- `/v1/runtime-config`
- `/v1/queue-backend`
