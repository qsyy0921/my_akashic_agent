# TDD: Agent Job External Lease Approval Preflight

## Service tests

- `TestAgentJobExternalLeasePreflightServiceBlocksWhenPlanNotReady`
- `TestAgentJobExternalLeasePreflightServiceRequiresApprovalAfterPlanReady`
- `TestAgentJobExternalLeasePreflightServiceAllowsApprovedReadyPlan`

## HTTP test

- `TestAgentJobExternalLeasePreflightEndpointReturnsApprovalBoundGate`

## Verifier tests

- `test_build_result_marks_live_verified_when_both_scenarios_pass`
- `test_build_result_marks_failure_when_preflight_check_fails`
- `test_build_result_marks_error_when_runtime_setup_raises`

## Live verification

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-agent-job-external-lease-cutover-preflight.ps1`
