from __future__ import annotations

from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify-go-migration-goal.ps1"
)


def test_goal_verifier_persists_default_json_artifact_and_current_state_snapshot():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    assert '[string]$JsonOutputPath = ""' in script_text
    assert '[ValidateSet("summary", "full", "none")][string]$StdoutMode = "summary"' in script_text
    assert "current_state = $currentState" in script_text
    assert '$JsonOutputPath = Join-Path $repoRoot ".codex-goal-verifier.json"' in script_text
    assert "Set-Content -LiteralPath $JsonOutputPath -Value $json -Encoding UTF8" in script_text


def test_goal_verifier_supports_summary_full_and_none_stdout_modes():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        'if ($StdoutMode -eq "none") {',
        'if ($StdoutMode -eq "full") {',
        '$stdoutSummary = [pscustomobject]@{',
        'status = "artifact_written"',
        "artifact_path = $JsonOutputPath",
        "migration_bucket_summary = $result.migration_bucket_summary",
        '$stdoutSummary | ConvertTo-Json -Depth 8',
    ):
        assert expected in script_text


def test_goal_verifier_current_state_includes_live_runtime_cutover_fields():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        "qq_group_send_enabled = $runtimeConfig.data.delivery.qq_group_send_enabled",
        "telegram_token_configured = $runtimeConfig.data.delivery.telegram_token_configured",
        "outbox_execution_owner = $queueBackend.data.outbox_execution_owner",
        "outbox_execution_scope = $queueBackend.data.outbox_execution_scope",
        "agent_job_external_lease_ready = $agentJobExternalLeaseReadiness.data.ready",
        "receiver_statuses_telegram = $receiverStatuses.data.totals.telegram",
        "receiver_statuses_stopped = $receiverStatuses.data.totals.stopped",
        "receiver_statuses_heartbeat_stale = @($receiverStatusCleanupVerification.live_runtime.before.stale_receivers).Count",
        "agent_workers_total = $agentWorkerStatuses.data.totals.workers",
        "agent_workers_stale = $agentWorkerStatuses.data.totals.stale",
        "agent_workers_stopped = $agentWorkerStatuses.data.totals.stopped",
        "runtime_overview_agent_workers_stale = $runtimeOverview.data.summary.agent_workers_stale",
        "knowledge_configured_rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.configured_rag_datasets",
        "knowledge_rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_datasets",
        "knowledge_rag_dataset_index_ready = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_ready",
        "dashboard_fallback_category = if ($dashboardReadModelChecks.proactive_tick_logs_readable -and",
    ):
        assert expected in script_text


def test_goal_verifier_exposes_machine_readable_migration_residuals():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        "migration_residuals = $migrationResiduals",
        "qq_napcat_cutover = [pscustomobject]@{",
        'category = "still_not_fully_cut_over"',
        "telegram_backend = [pscustomobject]@{",
        "agent_job_external_lease_result_ack = [pscustomobject]@{",
        "scheduler_runtime = [pscustomobject]@{",
        "knowledge_rag_state_boundary = [pscustomobject]@{",
        "worker_control_executors = [pscustomobject]@{",
        "media_recovery_private_source_executor = [pscustomobject]@{",
        "dashboard_read_models = [pscustomobject]@{",
        "mq_adapter_boundary = [pscustomobject]@{",
        "python_owned_surfaces = [pscustomobject]@{",
        "migration_bucket = \"explicitly_python_owned\"",
        "migration_bucket_summary = $migrationBucketSummary",
    ):
        assert expected in script_text


def test_goal_verifier_exposes_exact_bucket_names_for_residual_summary():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        "already_in_go_only_missing_live_verification = @(@(",
        "go_control_plane_present_but_real_executor_missing = @(@(",
        "still_not_fully_cut_over = @(@(",
        "explicitly_python_owned = @(@(",
    ):
        assert expected in script_text


def test_goal_verifier_wires_proactive_runtime_flow_smoke():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyProactiveRuntimeFlowLiveSmokePath = Join-Path $scriptsDir "verify-proactive-runtime-flow-live-smoke.ps1"',
        "$proactiveRuntimeFlowSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifyProactiveRuntimeFlowLiveSmokePath -Arguments @{",
        "proactive_flow_smoke = $proactiveRuntimeFlowSmokeVerification.conclusion",
        "proactive_runtime_flow_smoke = $proactiveRuntimeFlowSmokeVerification",
    ):
        assert expected in script_text


def test_goal_verifier_wires_receiver_status_cleanup_live_smoke():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyReceiverStatusCleanupPath = Join-Path $scriptsDir "verify-receiver-status-cleanup-live-smoke.ps1"',
        "$receiverStatusCleanupVerification = Invoke-RepoScriptJson -ScriptPath $verifyReceiverStatusCleanupPath -Arguments @{",
        "receiver_status_cleanup = $receiverStatusCleanupVerification",
        "receiver_status_cleanup = $receiverStatusCleanupVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_agent_worker_status_startup_prune_live_smoke():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyAgentWorkerStatusStartupPrunePath = Join-Path $scriptsDir "verify-agent-worker-status-startup-prune-live-smoke.ps1"',
        "$agentWorkerStatusStartupPruneVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusStartupPrunePath -Arguments @{",
        "agent_worker_status_startup_prune = $agentWorkerStatusStartupPruneVerification",
        "agent_worker_status_startup_prune = $agentWorkerStatusStartupPruneVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_agent_job_external_lease_launcher_preflight():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyAgentJobExternalLeaseLauncherPreflightPath = Join-Path $scriptsDir "verify-agent-job-external-lease-launcher-preflight.ps1"',
        "$agentJobExternalLeaseLauncherPreflightVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseLauncherPreflightPath -Arguments @{",
        "launcher_cutover_preflight = $agentJobExternalLeaseLauncherPreflightVerification.conclusion.category",
        '"launcher_cutover_preflight"',
        "agent_job_external_lease_launcher_preflight = $agentJobExternalLeaseLauncherPreflightVerification",
        "agent_job_external_lease_launcher_preflight = $agentJobExternalLeaseLauncherPreflightVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_agent_job_external_lease_launcher_bundle():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyAgentJobExternalLeaseLauncherBundlePath = Join-Path $scriptsDir "verify-agent-job-external-lease-launcher-bundle.ps1"',
        "$agentJobExternalLeaseLauncherBundleVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseLauncherBundlePath -Arguments @{",
        "launcher_bundle_live_smoke = $agentJobExternalLeaseLauncherBundleVerification.conclusion.category",
        '"launcher_bundle_live_smoke"',
        "agent_job_external_lease_launcher_bundle = $agentJobExternalLeaseLauncherBundleVerification",
        "agent_job_external_lease_launcher_bundle = $agentJobExternalLeaseLauncherBundleVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_agent_job_external_lease_cutover_diff():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyAgentJobExternalLeaseCutoverDiffPath = Join-Path $scriptsDir "verify-agent-job-external-lease-cutover-diff.ps1"',
        "$agentJobExternalLeaseCutoverDiffVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseCutoverDiffPath -Arguments @{",
        "cutover_diff_live_verifier = $agentJobExternalLeaseCutoverDiffVerification.conclusion.category",
        '"cutover_diff_live_verifier"',
        "agent_job_external_lease_cutover_diff = $agentJobExternalLeaseCutoverDiffVerification",
        "agent_job_external_lease_cutover_diff = $agentJobExternalLeaseCutoverDiffVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_media_asset_content_recovery_live_smoke():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyMediaAssetContentRecoveryLiveSmokePath = Join-Path $scriptsDir "verify-media-asset-content-recovery-live-smoke.ps1"',
        "$mediaAssetContentRecoveryLiveSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifyMediaAssetContentRecoveryLiveSmokePath -Arguments @{",
        'category = if ($mediaAssetContentRecoveryLiveSmokeVerification.conclusion.status -eq "live_verified") {',
        "http_https_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification.conclusion.category",
        "media_recovery_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification",
        "media_recovery_http_https_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_dashboard_media_asset_content_recovery_boundary_verifier():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyDashboardMediaAssetContentRecoveryBoundaryPath = Join-Path $scriptsDir "verify-dashboard-media-asset-content-recovery-boundary.ps1"',
        "$dashboardMediaAssetContentRecoveryBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardMediaAssetContentRecoveryBoundaryPath -Arguments @{",
        "media_asset_content_recovery_table = (",
        "$dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_summary_matches_runtime_media_asset_content_recovery",
        "dashboard_media_asset_content_recovery_boundary = $dashboardMediaAssetContentRecoveryBoundaryVerification",
        "dashboard_media_asset_content_recovery_boundary = $dashboardMediaAssetContentRecoveryBoundaryVerification.conclusion",
        "dashboard_live_category = $dashboardMediaAssetContentRecoveryBoundaryVerification.conclusion.category",
    ):
        assert expected in script_text


def test_goal_verifier_wires_dashboard_media_asset_content_boundary_verifier():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyDashboardMediaAssetContentBoundaryPath = Join-Path $scriptsDir "verify-dashboard-media-asset-content-boundary.ps1"',
        "$dashboardMediaAssetContentBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardMediaAssetContentBoundaryPath -Arguments @{",
        "media_asset_content_table = (",
        "$dashboardMediaAssetContentBoundaryVerification.checks.dashboard_overview_exposes_media_asset_content_card",
        "$dashboardMediaAssetContentBoundaryVerification.checks.dashboard_media_asset_content_preflight_proxy_matches_runtime",
        "dashboard_media_asset_content_boundary = $dashboardMediaAssetContentBoundaryVerification",
        "dashboard_media_asset_content_boundary = $dashboardMediaAssetContentBoundaryVerification.conclusion",
        "dashboard_media_asset_content_live_category = $dashboardMediaAssetContentBoundaryVerification.conclusion.category",
        "media asset content, media asset content recovery",
    ):
        assert expected in script_text


def test_goal_verifier_wires_dashboard_proactive_tick_logs_boundary_verifier():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyDashboardProactiveTickLogsBoundaryPath = Join-Path $scriptsDir "verify-dashboard-proactive-tick-logs-boundary.ps1"',
        "$dashboardProactiveTickLogsBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardProactiveTickLogsBoundaryPath -Arguments @{",
        "proactive_tick_logs_readable = ($proactiveVerification.checks.dashboard_tick_logs_readable -and $dashboardProactiveTickLogsBoundaryVerification.conclusion.status -eq \"live_verified\")",
        "dashboard_proactive_tick_logs_boundary = $dashboardProactiveTickLogsBoundaryVerification",
        "dashboard_proactive_tick_logs_boundary = $dashboardProactiveTickLogsBoundaryVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_wires_dashboard_knowledge_rag_state_boundary_verifier():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$verifyDashboardKnowledgeRagStateBoundaryPath = Join-Path $scriptsDir "verify-dashboard-knowledge-rag-state-boundary.ps1"',
        "$dashboardKnowledgeRagStateBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardKnowledgeRagStateBoundaryPath -Arguments @{",
        '$dashboardKnowledgeRagStateBoundaryVerification.conclusion.status -eq "live_verified"',
        "dashboard_knowledge_rag_state_boundary = $dashboardKnowledgeRagStateBoundaryVerification",
        "knowledge_rag_state_boundary = $dashboardKnowledgeRagStateBoundaryVerification.conclusion",
    ):
        assert expected in script_text


def test_goal_verifier_retries_dashboard_knowledge_rag_state_boundary_once_on_failure():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    for expected in (
        '$dashboardKnowledgeRagStateBoundaryVerification.conclusion.status -ne "live_verified"',
        "Start-Sleep -Seconds 2",
        "$dashboardKnowledgeRagStateBoundaryVerificationRetry = Invoke-RepoScriptJson -ScriptPath $verifyDashboardKnowledgeRagStateBoundaryPath -Arguments @{",
        '$dashboardKnowledgeRagStateBoundaryVerificationRetry.conclusion.status -eq "live_verified"',
        "$dashboardKnowledgeRagStateBoundaryVerification = $dashboardKnowledgeRagStateBoundaryVerificationRetry",
    ):
        assert expected in script_text


def test_goal_verifier_aliases_dashboard_read_models_under_checks():
    script_text = SCRIPT_PATH.read_text(encoding="utf-8")

    assert "checks = [pscustomobject]@{" in script_text
    assert "dashboard_read_models = $dashboardReadModelChecks" in script_text
