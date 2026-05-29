# Review: Phase 7.3 Image Job AgentJob Bridge

Spec:

- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`

Implementation summary:

- Kept the legacy `/v1/image-jobs` API and `ImageJob` response contract intact.
- Added an optional `AgentJobRepository` dependency to `ImageJobService`.
- Wired the runtime service so every new legacy image job also creates a
  same-ID generic `AgentJob` with `job_type=image_generation`.
- Stored image prompt, provider, model, size, count, request id, requester id,
  source event id, and legacy image job id in the generic job payload/metadata.
- Preserved the old constructor for tests and callers that do not need the
  compatibility bridge.

Tests run:

- Pending before commit: Go unit tests, Go build, targeted Python client
  regressions, SDD/shadow regressions, and `git diff --check`.

Findings:

- This is a bridge, not the final image worker migration. The Python image
  worker still needs to lease generic `image_generation` jobs and report
  completion back to both the generic job API and the legacy image-job API until
  the legacy API can be retired.

Decision:

- Accept the bridge because it lets Python workers start consuming the generic
  queue without breaking existing `/v1/image-jobs` callers.

Follow-ups:

- Add the Python image worker loop for `image_generation`.
- Add a compatibility completion helper that updates generic and legacy job
  states together.
- Later remove the image-specific queue after all callers use `AgentJob`.
