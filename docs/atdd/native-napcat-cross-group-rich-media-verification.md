# ATDD: Native NapCat Cross-Group Rich Media Verification

## Scope

- Confirms whether the current QQ rich-media blocker is limited to one group or
  applies across multiple observe-only groups under the same NapCat session.
- Confirms whether current Go outbox owner conclusions should remain
  `text_only`.

## Preconditions

- Docker Desktop running locally.
- NapCat container for QQ `1049511700` reachable on `ws://127.0.0.1:3001`.
- At least three enabled QQ observe-only groups for the same account.
- Existing baseline evidence from one prior native NapCat rich-media run.

## Scenarios

### Scenario 1

- Action:
  Run `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 3219982`.
- Expect:
  `image_upload` and `file_upload` return `retcode=0` with `file_path`, while
  final image/file send still returns `rich media transfer failed`.

### Scenario 2

- Action:
  Run `.\scripts\run-napcat-native-rich-media-smoke.ps1 -GroupId 164369633`.
- Expect:
  Staging still succeeds and final image/file send still returns the same
  platform error.

### Scenario 3

- Action:
  Re-read `/v1/runtime-config`, `/v1/outbound-cutover/readiness`,
  `/v1/receiver-statuses`, and knowledge planner status after the comparison.
- Expect:
  Text-only Go owner remains intact, Telegram still reports missing token if no
  token was added, and knowledge planner remains owned by Go without reviving
  Python legacy admission.

## Failure Signals

- Additional groups succeed while the original group still fails.
- Upload staging fails before the final QQ send step.
- Cross-group results diverge enough that session-wide classification is no
  longer justified.

## Evidence

- Script outputs for groups `3219982` and `164369633`.
- `/v1/runtime-config`
- `/v1/outbound-cutover/readiness`
- `/v1/receiver-statuses`
- `/v1/knowledge-job-planner/cutover-plan`
