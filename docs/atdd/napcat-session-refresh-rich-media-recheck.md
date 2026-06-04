# ATDD: NapCat Session Refresh Rich Media Recheck

## Scope

- Confirms whether a simple NapCat container restart can restore QQ rich-media
  send capability for the current session.
- Narrows the blocker to manual re-login or replacement when restart is
  insufficient.

## Preconditions

- Docker Desktop running locally.
- `napcat` container healthy enough to reconnect after restart.
- Real QQ observe-only group available for smoke, for example `3219982`.
- Existing native parity evidence showing `rich media transfer failed`.

## Scenarios

### Scenario 1

- Action:
  Restart the `napcat` container, then read `/v1/receiver-statuses`.
- Expect:
  QQ receiver reconnects and returns to `status=connected`.

### Scenario 2

- Action:
  After restart, run
  `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 3219982`.
- Expect:
  If image/file staging still succeeds but final send still returns
  `rich media transfer failed`, container restart is insufficient.

### Scenario 3

- Action:
  Re-read `/v1/runtime-config`, `/v1/outbound-cutover/readiness`, and
  `/v1/knowledge-job-planner/cutover-plan`.
- Expect:
  Text-only Go owner remains intact, Telegram still reports missing token if no
  token was added, and Go knowledge planner ownership remains unchanged.

## Failure Signals

- Container restart breaks receiver reconnect.
- Native smoke no longer reaches upload staging.
- Post-restart rich-media outcome becomes inconsistent or ambiguous.

## Evidence

- `docker restart napcat`
- `/v1/receiver-statuses`
- `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 3219982`
- `/v1/runtime-config`
- `/v1/outbound-cutover/readiness`
- `/v1/knowledge-job-planner/cutover-plan`
