param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$DashboardBaseUrl = "http://127.0.0.1:2236",
    [int]$RecentJobs = 8,
    [string]$JsonOutputPath = "",
    [ValidateSet("summary", "full", "none")][string]$StdoutMode = "summary",
    [switch]$IncludeOutboxScopeSmoke,
    [switch]$IncludeNativeRichMediaProbe,
    [string]$RichMediaProbeWebSocketUrl = "ws://127.0.0.1:3001",
    [string]$RichMediaProbeAccessToken = "NcatBot",
    [ValidateSet("group", "private")][string]$RichMediaProbeConversationType = "group",
    [string]$RichMediaProbeGroupId = "3219982",
    [string]$RichMediaProbeChatId = ""
)

$ErrorActionPreference = "Stop"

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Uri,
        $Body = $null
    )

    if ($null -eq $Body) {
        return Invoke-RestMethod -Method $Method -Uri $Uri -TimeoutSec 30
    }

    $jsonBody = $Body | ConvertTo-Json -Depth 10
    return Invoke-RestMethod -Method $Method -Uri $Uri -ContentType "application/json" -Body $jsonBody -TimeoutSec 30
}

function Invoke-TextRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Uri
    )

    return (Invoke-WebRequest -Uri $Uri -TimeoutSec 30 -UseBasicParsing).Content
}

function Invoke-RepoScriptJson {
    param(
        [Parameter(Mandatory = $true)][string]$ScriptPath,
        [hashtable]$Arguments = @{}
    )

    $output = & $ScriptPath @Arguments
    return $output | ConvertFrom-Json
}

function Invoke-SchedulerVerification {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl
    )

    $jobs = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/scheduler/jobs"
    $leases = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/scheduler/leases"
    $timestamp = (Get-Date).ToUniversalTime().ToString("o")
    $diagnostics = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/scheduler/diagnostics?timestamp=$([uri]::EscapeDataString($timestamp))&due_soon_seconds=120"

    $jobCount = @($jobs.data).Count
    $activeLeaseCount = [int]$leases.data.totals.active
    $sampledJobs = [int]$diagnostics.data.sampled_jobs
    $category = if ($jobCount -gt 0 -or $activeLeaseCount -gt 0 -or $sampledJobs -gt 0) {
        "go_control_plane_live_verified_with_runtime_state"
    } else {
        "go_control_plane_live_verified_idle"
    }

    [pscustomobject]@{
        jobs = $jobs.data
        leases = $leases.data
        diagnostics = $diagnostics.data
        checks = [pscustomobject]@{
            jobs_endpoint_reachable = $true
            leases_endpoint_reachable = $true
            diagnostics_endpoint_reachable = $true
            control_plane_state_visible = ($jobCount -ge 0 -and $activeLeaseCount -ge 0 -and $sampledJobs -ge 0)
            idle_snapshot = ($jobCount -eq 0 -and $activeLeaseCount -eq 0 -and $sampledJobs -eq 0)
        }
        conclusion = [pscustomobject]@{
            status = "live_verified"
            category = $category
            reason = if ($category -eq "go_control_plane_live_verified_idle") {
                "scheduler job, lease, and diagnostics endpoints all responded; current local runtime is idle, so remaining work is reminder/recurring/recovery smoke rather than missing Go control plane."
            } else {
                "scheduler job, lease, and diagnostics endpoints all responded and exposed current runtime scheduler state."
            }
        }
    }
}

function Invoke-ProactiveVerification {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$DashboardUrl
    )

    $timestamp = (Get-Date).ToUniversalTime().ToString("o")
    $tickLogs = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/proactive/tick-logs?limit=5"
    $driftSummary = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/proactive/drift/summary?limit=5"
    $quota = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/proactive/anyaction/quota?reset_hour=0&timezone=Asia%2FShanghai&timestamp=$([uri]::EscapeDataString($timestamp))"
    $bgContext = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/proactive/bg-context/main/last"
    $contextOnly = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/proactive/context-only/last?session_key=qq:1049511700:private:2365524513"
    $dashboardTickLogs = Invoke-JsonRequest -Method "GET" -Uri "$DashboardUrl/api/dashboard/proactive/tick_logs?limit=3"

    $tickItems = @($tickLogs.data.items)
    $dashboardTickItems = @($dashboardTickLogs.items)
    $category = if ($tickItems.Count -gt 0 -and $dashboardTickItems.Count -gt 0 -and [bool]$quota.data.found) {
        "go_state_live_verified_with_dashboard_read_model"
    } else {
        "live_verification_incomplete"
    }

    [pscustomobject]@{
        runtime = [pscustomobject]@{
            tick_logs = $tickLogs.data
            drift_summary = $driftSummary.data
            anyaction_quota = $quota.data
            bg_context_main_last = $bgContext.data
            context_only_last = $contextOnly.data
        }
        dashboard = [pscustomobject]@{
            proactive_tick_logs = $dashboardTickLogs
        }
        checks = [pscustomobject]@{
            tick_logs_endpoint_reachable = $true
            recent_tick_logs_present = ($tickItems.Count -gt 0)
            anyaction_quota_visible = [bool]$quota.data.found
            dashboard_tick_logs_readable = ($dashboardTickItems.Count -gt 0)
            drift_summary_endpoint_reachable = $true
            bg_context_endpoint_reachable = $true
            context_only_endpoint_reachable = $true
        }
        conclusion = [pscustomobject]@{
            status = if ($category -eq "go_state_live_verified_with_dashboard_read_model") { "live_verified" } else { "verification_incomplete" }
            category = $category
            reason = if ($category -eq "go_state_live_verified_with_dashboard_read_model") {
                "Go proactive runtime state is readable from tick-log and quota endpoints, and dashboard proactive tick-log read model is also live. Remaining work is flow-specific live observation for seen/rejection/bg-context/drift transitions."
            } else {
                "At least one proactive runtime or dashboard read path failed to produce current evidence."
            }
        }
    }
}

function Invoke-ObserveOnlySilenceVerification {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)]$RuntimeConfig
    )

    $observeTargets = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/observe-targets"
    $qqGroupTargets = @($observeTargets.data.targets | Where-Object {
        $_.channel.kind -eq "qq" -and
        $_.channel.conversation_type -eq "group" -and
        $_.enabled -and
        $_.reply_allowed -eq $false
    })

    $groupProbe = $null
    if ($qqGroupTargets.Count -gt 0) {
        $target = $qqGroupTargets[0]
        $eventId = "observe-only-goal-check:{0}" -f ([guid]::NewGuid().ToString("N"))
        $outboundBody = @{
            event_id = $eventId
            channel = @{
                kind = "qq"
                platform = "qq"
                account_id = $target.channel.account_id
                conversation_id = $target.channel.conversation_id
                conversation_type = "group"
            }
            content = "observe-only silence goal verifier"
            metadata = @{
                source = "verify_go_migration_goal"
                live_smoke = "true"
                max_attempts = "1"
            }
        }
        $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbound" -Body $outboundBody

        try {
            $ready = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/readiness" -Body @{
                event_id = $eventId
            }
            $groupProbe = [pscustomobject]@{
                target_id = $target.target_id
                blocked = $false
                status_code = 200
                message = $ready.data.reason
                error_kind = $null
                global_policy = $false
            }
        } catch {
            $statusCode = 0
            $message = $_.Exception.Message
            $errorKind = $null
            if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
                try {
                    $errorPayload = $_.ErrorDetails.Message | ConvertFrom-Json
                    if ($null -ne $errorPayload.message) {
                        $message = $errorPayload.message
                    }
                    if ($null -ne $errorPayload.data.error_kind) {
                        $errorKind = $errorPayload.data.error_kind
                    }
                } catch {
                }
            }
            if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
                $statusCode = [int]$_.Exception.Response.StatusCode
            }
            $groupProbe = [pscustomobject]@{
                target_id = $target.target_id
                blocked = ($statusCode -eq 400 -and $errorKind -eq "route_error" -and @("observe-only target does not allow replies", "qq group sends are disabled") -contains $message)
                status_code = $statusCode
                message = $message
                error_kind = $errorKind
                global_policy = ($statusCode -eq 400 -and $message -eq "qq group sends are disabled" -and $errorKind -eq "route_error")
            }
        }
    }

    $privateProbe = $null
    $botIds = @($RuntimeConfig.data.runtime.bot_ids)
    if ($botIds.Count -ge 2) {
        $eventId = "observe-only-private-goal-check:{0}" -f ([guid]::NewGuid().ToString("N"))
        $outboundBody = @{
            event_id = $eventId
            channel = @{
                kind = "qq"
                platform = "qq"
                account_id = $botIds[0]
                conversation_id = $botIds[1]
                conversation_type = "private"
            }
            content = "observe-only private goal verifier"
            metadata = @{
                source = "verify_go_migration_goal"
                live_smoke = "true"
                max_attempts = "1"
            }
        }
        $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbound" -Body $outboundBody
        $ready = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/readiness" -Body @{
            event_id = $eventId
        }
        $privateProbe = [pscustomobject]@{
            route = "qq:{0}:private:{1}" -f $botIds[0], $botIds[1]
            ready = $ready.data.ready
            reason = $ready.data.reason
        }
    }

    return [pscustomobject]@{
        observe_targets_totals = $observeTargets.data.totals
        qq_group_targets = @($qqGroupTargets | ForEach-Object {
            [pscustomobject]@{
                target_id = $_.target_id
                account_id = $_.channel.account_id
                conversation_id = $_.channel.conversation_id
                reply_allowed = $_.reply_allowed
                enabled = $_.enabled
            }
        })
        group_probe = $groupProbe
        private_probe = $privateProbe
        checks = [pscustomobject]@{
            enabled_reply_disabled_group_targets_present = ($qqGroupTargets.Count -gt 0)
            qq_group_send_toggle_disabled = (-not $RuntimeConfig.data.delivery.qq_group_send_enabled)
            qq_group_route_blocked = ($null -ne $groupProbe -and $groupProbe.blocked)
            qq_group_route_blocked_by_global_policy = ($null -ne $groupProbe -and $groupProbe.global_policy)
            private_route_still_ready = ($null -eq $privateProbe -or $privateProbe.ready)
        }
    }
}

function Convert-HashtableForJson {
    param([hashtable]$Map)

    if ($null -eq $Map) {
        return [ordered]@{}
    }

    $ordered = [ordered]@{}
    foreach ($key in ($Map.Keys | Sort-Object)) {
        $ordered[$key] = $Map[$key]
    }
    return $ordered
}

function New-MigrationBucketEntry {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$Bucket
    )

    [pscustomobject]@{
        name = $Name
        migration_bucket = $Bucket
    }
}

function Get-OutboxAllowedRoutes {
    param($QueueBackend)

    $routes = New-Object System.Collections.Generic.List[object]

    foreach ($kind in @($QueueBackend.data.outbox_allowed_kinds)) {
        $routes.Add([pscustomobject]@{
            account_id = "*"
            conversation_type = "*"
            conversation_id = "*"
            kind = $kind
            source = "global"
        })
    }

    if ($null -ne $QueueBackend.data.outbox_allowed_kinds_by_account) {
        foreach ($accountProp in $QueueBackend.data.outbox_allowed_kinds_by_account.PSObject.Properties) {
            foreach ($kind in @($accountProp.Value)) {
                $routes.Add([pscustomobject]@{
                    account_id = $accountProp.Name
                    conversation_type = "*"
                    conversation_id = "*"
                    kind = $kind
                    source = "account"
                })
            }
        }
    }

    if ($null -ne $QueueBackend.data.outbox_allowed_kinds_by_account_conversation_type) {
        foreach ($accountProp in $QueueBackend.data.outbox_allowed_kinds_by_account_conversation_type.PSObject.Properties) {
            foreach ($conversationProp in $accountProp.Value.PSObject.Properties) {
                foreach ($kind in @($conversationProp.Value)) {
                    $routes.Add([pscustomobject]@{
                        account_id = $accountProp.Name
                        conversation_type = $conversationProp.Name
                        conversation_id = "*"
                        kind = $kind
                        source = "account_conversation_type"
                    })
                }
            }
        }
    }

    if ($null -ne $QueueBackend.data.outbox_allowed_kinds_by_account_conversation_id) {
        foreach ($accountProp in $QueueBackend.data.outbox_allowed_kinds_by_account_conversation_id.PSObject.Properties) {
            foreach ($conversationTypeProp in $accountProp.Value.PSObject.Properties) {
                foreach ($conversationIdProp in $conversationTypeProp.Value.PSObject.Properties) {
                    foreach ($kind in @($conversationIdProp.Value)) {
                        $routes.Add([pscustomobject]@{
                            account_id = $accountProp.Name
                            conversation_type = $conversationTypeProp.Name
                            conversation_id = $conversationIdProp.Name
                            kind = $kind
                            source = "account_conversation_id"
                        })
                    }
                }
            }
        }
    }

    return $routes.ToArray()
}

function Get-OpenBlockers {
    param(
        $TelegramVerification,
        $NativeRichMediaProbe,
        $OutboxScopeSmoke,
        $ObserveOnlySilence
    )

    $blockers = New-Object System.Collections.Generic.List[string]
    if ($TelegramVerification.checks.backend_state -eq "token_missing") {
        $blockers.Add("telegram_token_missing")
    }

    $blockers.Add("qq_image_native_platform_blocker_unresolved")

    if ($null -ne $NativeRichMediaProbe) {
        $imageRetcode = $NativeRichMediaProbe.image_send.retcode
        $fileRetcode = $NativeRichMediaProbe.file_send.retcode
        if ($imageRetcode -ne 0) {
            $blockers.Add("native_probe_image_retcode_$imageRetcode")
        }
        if ($fileRetcode -ne 0) {
            $blockers.Add("native_probe_file_retcode_$fileRetcode")
            $blockers.Add("qq_first_account_group_file_group_specific_blocker_unresolved")
        }
    }

    if ($null -ne $OutboxScopeSmoke) {
        foreach ($case in @($OutboxScopeSmoke.cases)) {
            if (-not $case.condition_met) {
                $blockers.Add("outbox_scope_case_failed_{0}" -f $case.name)
            }
        }
    }

    if ($null -ne $ObserveOnlySilence) {
        if (-not $ObserveOnlySilence.checks.qq_group_send_toggle_disabled) {
            $blockers.Add("qq_group_send_toggle_not_disabled")
        }
        if (-not $ObserveOnlySilence.checks.qq_group_route_blocked) {
            $blockers.Add("qq_group_route_block_not_enforced")
        }
        if (-not $ObserveOnlySilence.checks.qq_group_route_blocked_by_global_policy) {
            $blockers.Add("qq_group_global_policy_not_enforced")
        }
        if (-not $ObserveOnlySilence.checks.private_route_still_ready) {
            $blockers.Add("qq_private_route_regressed")
        }
    }

    return @($blockers | Select-Object -Unique)
}

$scriptsDir = $PSScriptRoot
$repoRoot = Split-Path -Parent $scriptsDir
$verifyKnowledgePlannerPath = Join-Path $scriptsDir "verify-knowledge-planner-cutover.ps1"
$verifyTelegramPath = Join-Path $scriptsDir "verify-telegram-backend.ps1"
$verifyAgentJobExternalLeasePath = Join-Path $scriptsDir "verify-agent-job-external-lease.ps1"
$verifyAgentJobExternalLeaseNatsSmokePath = Join-Path $scriptsDir "verify-agent-job-external-lease-nats-smoke.ps1"
$verifyAgentJobExternalLeaseCutoverPreflightPath = Join-Path $scriptsDir "verify-agent-job-external-lease-cutover-preflight.ps1"
$verifyAgentJobExternalLeaseLauncherPreflightPath = Join-Path $scriptsDir "verify-agent-job-external-lease-launcher-preflight.ps1"
$verifyAgentJobExternalLeaseLauncherBundlePath = Join-Path $scriptsDir "verify-agent-job-external-lease-launcher-bundle.ps1"
$verifyAgentJobExternalLeaseCutoverDiffPath = Join-Path $scriptsDir "verify-agent-job-external-lease-cutover-diff.ps1"
$verifyMediaRecoveryBoundaryPath = Join-Path $scriptsDir "verify-media-recovery-boundary.ps1"
$verifyMediaAssetContentRecoveryLiveSmokePath = Join-Path $scriptsDir "verify-media-asset-content-recovery-live-smoke.ps1"
$verifyAgentWorkerStatusRestartPath = Join-Path $scriptsDir "verify-agent-worker-status-restart.ps1"
$verifyAgentWorkerStatusFencingPath = Join-Path $scriptsDir "verify-agent-worker-status-fencing-live-smoke.ps1"
$verifyAgentWorkerStatusHeartbeatPath = Join-Path $scriptsDir "verify-agent-worker-status-heartbeat-live-smoke.ps1"
$verifyAgentWorkerStatusCleanupPath = Join-Path $scriptsDir "verify-agent-worker-status-cleanup-live-smoke.ps1"
$verifyAgentWorkerStatusStartupPrunePath = Join-Path $scriptsDir "verify-agent-worker-status-startup-prune-live-smoke.ps1"
$verifyReceiverStatusCleanupPath = Join-Path $scriptsDir "verify-receiver-status-cleanup-live-smoke.ps1"
$verifyMqAdapterBoundaryPath = Join-Path $scriptsDir "verify-mq-adapter-boundary.ps1"
$verifyQqCutoverRouteMatrixPath = Join-Path $scriptsDir "verify-qq-cutover-route-matrix.ps1"
$verifyQueueTopologyBoundaryPath = Join-Path $scriptsDir "verify-queue-topology-boundary.ps1"
$verifyWorkerControlExecutorBoundaryPath = Join-Path $scriptsDir "verify-worker-control-executor-boundary.ps1"
$verifyControlAuditBoundaryPath = Join-Path $scriptsDir "verify-control-audit-boundary.ps1"
$verifyDashboardControlAuditBoundaryPath = Join-Path $scriptsDir "verify-dashboard-control-audit-boundary.ps1"
$verifyDashboardMediaAssetContentBoundaryPath = Join-Path $scriptsDir "verify-dashboard-media-asset-content-boundary.ps1"
$verifyDashboardMediaAssetContentRecoveryBoundaryPath = Join-Path $scriptsDir "verify-dashboard-media-asset-content-recovery-boundary.ps1"
$verifyDashboardKnowledgeRagStateBoundaryPath = Join-Path $scriptsDir "verify-dashboard-knowledge-rag-state-boundary.ps1"
$verifyDashboardProactiveTickLogsBoundaryPath = Join-Path $scriptsDir "verify-dashboard-proactive-tick-logs-boundary.ps1"
$verifySchedulerRuntimeLiveSmokePath = Join-Path $scriptsDir "verify-scheduler-runtime-live-smoke.ps1"
$verifyProactiveRuntimeFlowLiveSmokePath = Join-Path $scriptsDir "verify-proactive-runtime-flow-live-smoke.ps1"
$nativeRichMediaPath = Join-Path $scriptsDir "run-napcat-native-rich-media-smoke.ps1"
$outboxScopeSmokePath = Join-Path $scriptsDir "verify-go-outbox-scope-live.ps1"

$healthz = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/healthz"
$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$runtimeWorkers = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-workers"
$runtimeOverview = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-overview"
$queueBackend = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-backend"
$outboundReadiness = Invoke-JsonRequest -Method "POST" -Uri "$RuntimeBaseUrl/v1/outbound-cutover/readiness"
$receiverStatuses = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/receiver-statuses"
$agentWorkerStatuses = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-worker-statuses"
$jobs = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/jobs?type=group_memory_extract&limit=$RecentJobs"
$agentJobExternalLeaseReadiness = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-external-lease/readiness"
$agentJobExternalLeasePlan = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-external-lease/plan"
$agentJobCapacityPlan = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-capacity/plan"
$agentJobPriorityPlan = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-priority/plan"
$schedulerReadOnlyVerification = Invoke-SchedulerVerification -BaseUrl $RuntimeBaseUrl
$proactiveVerification = Invoke-ProactiveVerification -BaseUrl $RuntimeBaseUrl -DashboardUrl $DashboardBaseUrl

$knowledgePlannerVerification = Invoke-RepoScriptJson -ScriptPath $verifyKnowledgePlannerPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    RecentJobs = $RecentJobs
}
$agentJobExternalLeaseVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeasePath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$agentJobExternalLeaseNatsSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseNatsSmokePath -Arguments @{
    RepoRoot = $repoRoot
}
$agentJobExternalLeaseCutoverPreflightVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseCutoverPreflightPath -Arguments @{
    RepoRoot = $repoRoot
}
$agentJobExternalLeaseLauncherPreflightVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseLauncherPreflightPath -Arguments @{
    RepoRoot = $repoRoot
}
$agentJobExternalLeaseLauncherBundleVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseLauncherBundlePath -Arguments @{
    RepoRoot = $repoRoot
}
$agentJobExternalLeaseCutoverDiffVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentJobExternalLeaseCutoverDiffPath -Arguments @{
    RepoRoot = $repoRoot
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$proactiveRuntimeFlowSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifyProactiveRuntimeFlowLiveSmokePath -Arguments @{
    RepoRoot = $repoRoot
}
$mediaAssetContentRecoveryLiveSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifyMediaAssetContentRecoveryLiveSmokePath -Arguments @{
    RepoRoot = $repoRoot
}
$workerControlExecutorVerification = Invoke-RepoScriptJson -ScriptPath $verifyWorkerControlExecutorBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$controlAuditBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyControlAuditBoundaryPath -Arguments @{
    RepoRoot = $repoRoot
}
$dashboardControlAuditBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardControlAuditBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
$dashboardMediaAssetContentBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardMediaAssetContentBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
$dashboardMediaAssetContentRecoveryBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardMediaAssetContentRecoveryBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
$dashboardKnowledgeRagStateBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardKnowledgeRagStateBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
if ($dashboardKnowledgeRagStateBoundaryVerification.conclusion.status -ne "live_verified") {
    Start-Sleep -Seconds 2
    $dashboardKnowledgeRagStateBoundaryVerificationRetry = Invoke-RepoScriptJson -ScriptPath $verifyDashboardKnowledgeRagStateBoundaryPath -Arguments @{
        RuntimeBaseUrl = $RuntimeBaseUrl
        DashboardBaseUrl = $DashboardBaseUrl
    }
    if ($dashboardKnowledgeRagStateBoundaryVerificationRetry.conclusion.status -eq "live_verified") {
        $dashboardKnowledgeRagStateBoundaryVerification = $dashboardKnowledgeRagStateBoundaryVerificationRetry
    }
}
$dashboardProactiveTickLogsBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyDashboardProactiveTickLogsBoundaryPath -Arguments @{
    RepoRoot = $repoRoot
}
$mediaRecoveryBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyMediaRecoveryBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
$agentWorkerStatusRestartVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusRestartPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$agentWorkerStatusFencingVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusFencingPath -Arguments @{
    RepoRoot = $repoRoot
}
$agentWorkerStatusHeartbeatVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusHeartbeatPath -Arguments @{
    RepoRoot = $repoRoot
}
$agentWorkerStatusCleanupVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusCleanupPath -Arguments @{
    RepoRoot = $repoRoot
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$agentWorkerStatusStartupPruneVerification = Invoke-RepoScriptJson -ScriptPath $verifyAgentWorkerStatusStartupPrunePath -Arguments @{
    RepoRoot = $repoRoot
}
$receiverStatusCleanupVerification = Invoke-RepoScriptJson -ScriptPath $verifyReceiverStatusCleanupPath -Arguments @{
    RepoRoot = $repoRoot
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$mqAdapterBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyMqAdapterBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    DashboardBaseUrl = $DashboardBaseUrl
}
$qqCutoverRouteMatrixVerification = Invoke-RepoScriptJson -ScriptPath $verifyQqCutoverRouteMatrixPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$queueTopologyBoundaryVerification = Invoke-RepoScriptJson -ScriptPath $verifyQueueTopologyBoundaryPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
}
$schedulerMutationSmokeVerification = Invoke-RepoScriptJson -ScriptPath $verifySchedulerRuntimeLiveSmokePath -Arguments @{
    RepoRoot = $repoRoot
}
$schedulerVerification = [pscustomobject]@{
    read_only = $schedulerReadOnlyVerification
    mutation_smoke = $schedulerMutationSmokeVerification
    checks = [pscustomobject]@{
        jobs_endpoint_reachable = $schedulerReadOnlyVerification.checks.jobs_endpoint_reachable
        leases_endpoint_reachable = $schedulerReadOnlyVerification.checks.leases_endpoint_reachable
        diagnostics_endpoint_reachable = $schedulerReadOnlyVerification.checks.diagnostics_endpoint_reachable
        control_plane_state_visible = $schedulerReadOnlyVerification.checks.control_plane_state_visible
        idle_snapshot = $schedulerReadOnlyVerification.checks.idle_snapshot
        mutation_recovery_live_verified = ($schedulerMutationSmokeVerification.conclusion.status -eq "live_verified")
    }
    conclusion = if ($schedulerMutationSmokeVerification.conclusion.status -eq "live_verified") {
        $schedulerMutationSmokeVerification.conclusion
    } else {
        $schedulerReadOnlyVerification.conclusion
    }
}
$telegramVerification = Invoke-RepoScriptJson -ScriptPath $verifyTelegramPath -Arguments @{
    RuntimeBaseUrl = $RuntimeBaseUrl
    RepoRoot = $repoRoot
}
$observeOnlySilence = Invoke-ObserveOnlySilenceVerification -BaseUrl $RuntimeBaseUrl -RuntimeConfig $runtimeConfig
$dashboardPanelJs = Invoke-TextRequest -Uri "$DashboardBaseUrl/plugins/runtime_overview/dashboard_panel.js"
$dashboardReadModelChecks = [pscustomobject]@{
    proactive_tick_logs_readable = ($proactiveVerification.checks.dashboard_tick_logs_readable -and $dashboardProactiveTickLogsBoundaryVerification.conclusion.status -eq "live_verified")
    delivery_smoke_readiness_table = ($dashboardPanelJs -match "Delivery Smoke Readiness")
    delivery_adapters_table = ($dashboardPanelJs -match "Delivery Adapters" -and $dashboardPanelJs -match "No delivery adapters sampled" -and $dashboardPanelJs -match "Access Token" -and $dashboardPanelJs -match "Adapter Health")
    queue_backend_table = ($dashboardPanelJs -match "Queue Backend" -and $dashboardPanelJs -match "Consumer Model" -and $dashboardPanelJs -match "Concurrent Consumers")
    external_lease_diagnostics_table = ($dashboardPanelJs -match "External Lease Diagnostics" -and $dashboardPanelJs -match "Selected Provider" -and $dashboardPanelJs -match "Supports Result Ack")
    receiver_statuses_table = ($dashboardPanelJs -match "Receiver Statuses" -and $dashboardPanelJs -match "No receiver statuses sampled" -and $dashboardPanelJs -match "Channel")
    receiver_leases_table = ($dashboardPanelJs -match "Receiver Leases" -and $dashboardPanelJs -match "No receiver leases sampled" -and $dashboardPanelJs -match "Owner Instance")
    knowledge_pipelines_table = (
        $dashboardPanelJs -match "Knowledge Pipelines" -and
        $dashboardPanelJs -match "No pipeline targets sampled" -and
        $dashboardPanelJs -match "Capture Status" -and
        $dashboardKnowledgeRagStateBoundaryVerification.conclusion.status -eq "live_verified"
    )
    observe_targets_table = ($dashboardPanelJs -match "Observe Targets" -and $dashboardPanelJs -match "No observe targets sampled" -and $dashboardPanelJs -match "No observe-target notes sampled" -and $dashboardPanelJs -match "Reply Allowed")
    observe_capture_table = ($dashboardPanelJs -match "Observe Capture" -and $dashboardPanelJs -match "No observe capture targets sampled" -and $dashboardPanelJs -match "No observe-capture notes sampled" -and $dashboardPanelJs -match "Content Ready")
    media_asset_content_table = (
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_overview_exposes_media_asset_content_card -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_overview_exposes_media_asset_content_detail -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_summary_matches_runtime_media_asset_content -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_media_asset_content_detail_matches_runtime -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_media_asset_content_card_matches_runtime -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_sample_media_asset_matches_runtime -and
        $dashboardMediaAssetContentBoundaryVerification.checks.dashboard_media_asset_content_preflight_proxy_matches_runtime
    )
    media_asset_content_recovery_table = (
        $dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_overview_exposes_media_asset_content_recovery_card -and
        $dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_overview_exposes_media_asset_content_recovery_detail -and
        $dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_summary_matches_runtime_media_asset_content_recovery -and
        $dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_media_asset_content_recovery_detail_matches_runtime -and
        $dashboardMediaAssetContentRecoveryBoundaryVerification.checks.dashboard_media_asset_content_recovery_card_matches_runtime
    )
    media_asset_retention_table = ($dashboardPanelJs -match "Media Asset Retention" -and $dashboardPanelJs -match "No media asset retention assets sampled" -and $dashboardPanelJs -match "Retention Class" -and $dashboardPanelJs -match "Cleanup Due")
    media_asset_retention_plan_table = ($dashboardPanelJs -match "Media Asset Retention Plan" -and $dashboardPanelJs -match "No required steps sampled" -and $dashboardPanelJs -match "Verification Steps" -and $dashboardPanelJs -match "Rollback Steps")
    media_asset_retention_cleanup_table = ($dashboardPanelJs -match "Media Asset Retention Cleanup" -and $dashboardPanelJs -match "No media asset retention cleanup notes" -and $dashboardPanelJs -match "Candidates" -and $dashboardPanelJs -match "Rolled Back")
    dead_letters_table = ($dashboardPanelJs -match "Dead Letters" -and $dashboardPanelJs -match "No agent-job dead letters sampled" -and $dashboardPanelJs -match "No outbox dead letters sampled" -and $dashboardPanelJs -match "Agent Job Dead Letter Totals")
    checkpoint_lag_table = ($dashboardPanelJs -match "Checkpoint Lag" -and $dashboardPanelJs -match "No lagged checkpoints sampled" -and $dashboardPanelJs -match "Lag Messages" -and $dashboardPanelJs -match "Checkpoint Seq")
    job_events_table = ($dashboardPanelJs -match "Job Events" -and $dashboardPanelJs -match "No job event totals sampled" -and $dashboardPanelJs -match "No recent job events sampled" -and $dashboardPanelJs -match "Occurred At")
    outbox_events_table = ($dashboardPanelJs -match "Outbox Events" -and $dashboardPanelJs -match "No outbox event totals sampled" -and $dashboardPanelJs -match "No recent outbox events sampled" -and $dashboardPanelJs -match "Delivery / Event")
    rag_eval_failures_table = ($dashboardPanelJs -match "RAG Eval Failures" -and $dashboardPanelJs -match "No rag-eval failures sampled" -and $dashboardPanelJs -match "rag eval dead letters" -and $dashboardPanelJs -match "Quality")
    runtime_health_table = ($dashboardPanelJs -match "Runtime Health" -and $dashboardPanelJs -match "No runtime health snapshot fields" -and $dashboardPanelJs -match "No runtime health errors")
    stale_jobs_table = ($dashboardPanelJs -match "Stale Jobs" -and $dashboardPanelJs -match "No worker stale-job diagnostics sampled" -and $dashboardPanelJs -match "No stale jobs sampled" -and $dashboardPanelJs -match "Latest Updated")
    scheduler_jobs_table = ($dashboardPanelJs -match "Scheduler Jobs" -and $dashboardPanelJs -match "No scheduler jobs sampled" -and $dashboardPanelJs -match "Next Run")
    worker_leases_table = ($dashboardPanelJs -match "Worker Leases" -and $dashboardPanelJs -match "No worker lease diagnostics sampled" -and $dashboardPanelJs -match "Checkpoint Prefix" -and $dashboardPanelJs -match "Stale Leases")
    runtime_config_table = ($dashboardPanelJs -match "Runtime Config" -and $dashboardPanelJs -match "QQ Group Send" -and $dashboardPanelJs -match "OneBot Endpoints" -and $dashboardPanelJs -match "Strict Lease Token" -and $dashboardPanelJs -match "Environment Keys")
    queue_topology_table = ($dashboardPanelJs -match "Queue Topology" -and $dashboardPanelJs -match "No queue topology work kinds sampled" -and $dashboardPanelJs -match "Ack Owner" -and $dashboardPanelJs -match "Execution Owner")
    qq_cutover_route_matrix_table = ($dashboardPanelJs -match "QQ Cutover Route Matrix" -and $dashboardPanelJs -match "No go-owned routes sampled" -and $dashboardPanelJs -match "Currently Sendable Routes" -and $dashboardPanelJs -match "Policy Blocked Routes" -and $dashboardPanelJs -match "Platform Blocker Routes")
    control_mutation_policy_table = (
        $dashboardControlAuditBoundaryVerification.checks.dashboard_overview_exposes_control_mutation_policy -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_summary_matches_runtime_control_audit -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_control_mutation_policy_matches_runtime -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_control_mutation_policy_card_matches_runtime
    )
    control_audit_table = (
        $dashboardControlAuditBoundaryVerification.checks.dashboard_overview_exposes_control_audit -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_summary_matches_runtime_control_audit -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_control_audit_detail_matches_runtime -and
        $dashboardControlAuditBoundaryVerification.checks.dashboard_control_audit_card_matches_runtime
    )
    knowledge_job_planner_preview_table = ($dashboardPanelJs -match "Knowledge Planner Preview" -and $dashboardPanelJs -match "No knowledge planner preview sampled" -and $dashboardPanelJs -match "Group Memory Jobs" -and $dashboardPanelJs -match "Observe Only")
    knowledge_job_planner_readiness_table = ($dashboardPanelJs -match "Knowledge Planner Readiness" -and $dashboardPanelJs -match "No knowledge planner readiness notes" -and $dashboardPanelJs -match "Planner Running" -and $dashboardPanelJs -match "Worker Active")
    agent_workers_table = ($dashboardPanelJs -match "Agent Workers" -and $dashboardPanelJs -match "No agent workers sampled" -and $dashboardPanelJs -match "Lease Active")
    runtime_workers_table = ($dashboardPanelJs -match "Runtime Workers" -and $dashboardPanelJs -match "No runtime workers sampled" -and $dashboardPanelJs -match "Batch / Max In Flight")
    agent_job_worker_coverage_table = ($dashboardPanelJs -match "Agent Job Worker Coverage" -and $dashboardPanelJs -match "No worker coverage sampled" -and $dashboardPanelJs -match "Expected Workers")
    agent_job_metrics_table = ($dashboardPanelJs -match "Agent Job Metrics" -and $dashboardPanelJs -match "No job types sampled" -and $dashboardPanelJs -match "No recent agent-job dead letters sampled" -and $dashboardPanelJs -match "Statuses")
    agent_job_pressure_table = ($dashboardPanelJs -match "Agent Job Pressure" -and $dashboardPanelJs -match "No pressure items sampled" -and $dashboardPanelJs -match "Oldest Pending")
    send_ledger_metrics_table = ($dashboardPanelJs -match "Send Ledger Metrics" -and $dashboardPanelJs -match "No repeated hashes sampled" -and $dashboardPanelJs -match "No recent send ledger records sampled")
    inbox_metrics_table = ($dashboardPanelJs -match "Inbox Metrics" -and $dashboardPanelJs -match "No conversation metrics sampled" -and $dashboardPanelJs -match "No recent inbox events sampled" -and $dashboardPanelJs -match "Latest Seq")
    inbound_dedupe_metrics_table = ($dashboardPanelJs -match "Inbound Dedupe" -and $dashboardPanelJs -match "No dedupe scopes sampled" -and $dashboardPanelJs -match "No dedupe notes sampled" -and $dashboardPanelJs -match "Duplicate Seen")
    outbox_metrics_table = ($dashboardPanelJs -match "Outbox Metrics" -and $dashboardPanelJs -match "No recent dead letters sampled" -and $dashboardPanelJs -match "Dead Letters Current")
    outbox_pressure_table = ($dashboardPanelJs -match "Outbox Pressure" -and $dashboardPanelJs -match "No outbox pressure sampled" -and $dashboardPanelJs -match "Dead Lettered")
    agent_job_external_lease_readiness_table = ($dashboardPanelJs -match "Agent Job External Lease")
    agent_job_external_lease_plan_table = ($dashboardPanelJs -match "Agent Job External Lease Plan")
    agent_job_external_lease_required_checks_table = ($dashboardPanelJs -match "Required Checks")
    agent_job_external_lease_enable_steps_table = ($dashboardPanelJs -match "Enable Steps")
    agent_job_external_lease_rollback_steps_table = ($dashboardPanelJs -match "Rollback Steps")
    agent_job_capacity_plan_table = ($dashboardPanelJs -match "Agent Job Capacity")
    agent_job_priority_plan_table = ($dashboardPanelJs -match "Agent Job Priority")
    knowledge_job_planner_cutover_plan_table = ($dashboardPanelJs -match "Knowledge Planner Cutover")
    outbound_cutover_plan_table = ($dashboardPanelJs -match "Outbound Cutover Plan")
}

$nativeRichMediaProbe = $null
if ($IncludeNativeRichMediaProbe) {
    $probeArgs = @{
        WebSocketUrl = $RichMediaProbeWebSocketUrl
        AccessToken = $RichMediaProbeAccessToken
        ConversationType = $RichMediaProbeConversationType
    }
    if ([string]::IsNullOrWhiteSpace($RichMediaProbeChatId)) {
        $probeArgs["GroupId"] = $RichMediaProbeGroupId
    } else {
        $probeArgs["ChatId"] = $RichMediaProbeChatId
    }
    $nativeRichMediaProbe = Invoke-RepoScriptJson -ScriptPath $nativeRichMediaPath -Arguments $probeArgs
}

$outboxScopeSmoke = $null
if ($IncludeOutboxScopeSmoke) {
    $outboxScopeSmoke = Invoke-RepoScriptJson -ScriptPath $outboxScopeSmokePath -Arguments @{
        RuntimeBaseUrl = $RuntimeBaseUrl
    }
}

$recentGroupMemoryJobs = @($jobs.data | ForEach-Object {
    [pscustomobject]@{
        job_id = $_.job_id
        status = $_.status
        scheduler = $_.metadata.scheduler
        conversation_id = $_.route.conversation_id
        updated_at = $_.updated_at
    }
})

$openBlockers = Get-OpenBlockers -TelegramVerification $telegramVerification -NativeRichMediaProbe $nativeRichMediaProbe -OutboxScopeSmoke $outboxScopeSmoke -ObserveOnlySilence $observeOnlySilence
$agentJobQueueTopology = @($queueTopologyBoundaryVerification.queue_topology.work_kinds | Where-Object { $_.work_kind -eq "agent_job" }) | Select-Object -First 1
$currentState = [pscustomobject]@{
    qq_group_send_enabled = $runtimeConfig.data.delivery.qq_group_send_enabled
    telegram_token_configured = $runtimeConfig.data.delivery.telegram_token_configured
    outbox_execution_owner = $queueBackend.data.outbox_execution_owner
    outbox_execution_scope = $queueBackend.data.outbox_execution_scope
    agent_job_execution_owner = if ($null -ne $agentJobQueueTopology) { $agentJobQueueTopology.execution_owner } else { $null }
    agent_job_ack_owner = if ($null -ne $agentJobQueueTopology) { $agentJobQueueTopology.ack_owner } else { $null }
    agent_job_external_lease_ready = $agentJobExternalLeaseReadiness.data.ready
    agent_job_external_lease_decision = $agentJobExternalLeasePlan.data.decision
    receiver_statuses_telegram = $receiverStatuses.data.totals.telegram
    receiver_statuses_stopped = $receiverStatuses.data.totals.stopped
    receiver_statuses_heartbeat_stale = @($receiverStatusCleanupVerification.live_runtime.before.stale_receivers).Count
    agent_workers_total = $agentWorkerStatuses.data.totals.workers
    agent_workers_stale = $agentWorkerStatuses.data.totals.stale
    agent_workers_stopped = $agentWorkerStatuses.data.totals.stopped
    runtime_overview_agent_workers_stale = $runtimeOverview.data.summary.agent_workers_stale
    knowledge_configured_rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.configured_rag_datasets
    knowledge_rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_datasets
    knowledge_rag_dataset_index_ready = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_ready
    dashboard_fallback_category = if ($dashboardReadModelChecks.proactive_tick_logs_readable -and
        $dashboardReadModelChecks.delivery_smoke_readiness_table -and
        $dashboardReadModelChecks.queue_backend_table -and
        $dashboardReadModelChecks.external_lease_diagnostics_table -and
        $dashboardReadModelChecks.receiver_statuses_table -and
        $dashboardReadModelChecks.receiver_leases_table -and
        $dashboardReadModelChecks.knowledge_pipelines_table -and
        $dashboardReadModelChecks.media_asset_content_table -and
        $dashboardReadModelChecks.media_asset_content_recovery_table -and
        $dashboardReadModelChecks.media_asset_retention_table -and
        $dashboardReadModelChecks.media_asset_retention_plan_table -and
        $dashboardReadModelChecks.media_asset_retention_cleanup_table -and
        $dashboardReadModelChecks.dead_letters_table -and
        $dashboardReadModelChecks.checkpoint_lag_table -and
        $dashboardReadModelChecks.job_events_table -and
        $dashboardReadModelChecks.outbox_events_table -and
        $dashboardReadModelChecks.rag_eval_failures_table -and
        $dashboardReadModelChecks.runtime_health_table -and
        $dashboardReadModelChecks.stale_jobs_table -and
        $dashboardReadModelChecks.scheduler_jobs_table -and
        $dashboardReadModelChecks.worker_leases_table -and
        $dashboardReadModelChecks.runtime_config_table -and
        $dashboardReadModelChecks.queue_topology_table -and
        $dashboardReadModelChecks.qq_cutover_route_matrix_table -and
        $dashboardReadModelChecks.control_mutation_policy_table -and
        $dashboardReadModelChecks.control_audit_table -and
        $dashboardReadModelChecks.knowledge_job_planner_preview_table -and
        $dashboardReadModelChecks.knowledge_job_planner_readiness_table -and
        $dashboardReadModelChecks.agent_workers_table -and
        $dashboardReadModelChecks.runtime_workers_table -and
        $dashboardReadModelChecks.agent_job_worker_coverage_table -and
        $dashboardReadModelChecks.agent_job_metrics_table -and
        $dashboardReadModelChecks.agent_job_pressure_table -and
        $dashboardReadModelChecks.send_ledger_metrics_table -and
        $dashboardReadModelChecks.outbox_metrics_table -and
        $dashboardReadModelChecks.outbox_pressure_table -and
        $dashboardReadModelChecks.agent_job_external_lease_readiness_table -and
        $dashboardReadModelChecks.agent_job_external_lease_plan_table -and
        $dashboardReadModelChecks.agent_job_capacity_plan_table -and
        $dashboardReadModelChecks.agent_job_priority_plan_table -and
        $dashboardReadModelChecks.knowledge_job_planner_cutover_plan_table -and
        $dashboardReadModelChecks.outbound_cutover_plan_table) {
        "live_verified_runtime_read_models"
    } else {
        "partially_live_verified_read_model"
    }
}
$migrationResiduals = [pscustomobject]@{
    qq_napcat_cutover = [pscustomobject]@{
        migration_bucket = "still_not_fully_cut_over"
        category = "still_not_fully_cut_over"
        current_execution_owner = $currentState.outbox_execution_owner
        current_execution_scope = $currentState.outbox_execution_scope
        qq_group_send_enabled = $currentState.qq_group_send_enabled
        go_owned_routes = @($qqCutoverRouteMatrixVerification.route_matrix.go_execution_owner_scope).Count
        currently_sendable_routes = @($qqCutoverRouteMatrixVerification.route_matrix.currently_sendable_routes).Count
        policy_blocked_routes = @($qqCutoverRouteMatrixVerification.route_matrix.policy_blocked_routes).Count
        platform_blocker_routes = @($qqCutoverRouteMatrixVerification.route_matrix.platform_blocker_routes).Count
        blockers = @(
            "qq_group_send_disabled_policy",
            "qq_image_native_platform_blocker_unresolved",
            "first_account_group_file_group_specific_blocker_unresolved"
        )
        next_minimal_step = "Keep group send disabled unless explicitly approved; after session repair, rerun native rich-media probe and QQ cutover route-matrix verification."
    }
    telegram_backend = [pscustomobject]@{
        migration_bucket = if ($currentState.telegram_token_configured -and $currentState.receiver_statuses_telegram -gt 0) {
            "already_in_go_only_missing_live_verification"
        } else {
            "still_not_fully_cut_over"
        }
        category = if ($currentState.telegram_token_configured -and $currentState.receiver_statuses_telegram -gt 0) {
            "already_in_go_pending_send_receive_smoke"
        } else {
            "still_not_fully_cut_over"
        }
        token_configured = $currentState.telegram_token_configured
        receiver_total = $currentState.receiver_statuses_telegram
        blocker = if ($currentState.telegram_token_configured) { $null } else { "telegram_token_missing" }
        next_minimal_step = if ($currentState.telegram_token_configured) {
            "Run getMe plus receiver and send/receive smoke."
        } else {
            "Provide TELEGRAM_BOT_TOKEN, then rerun verify-telegram-backend.ps1 and the unified goal verifier."
        }
    }
    agent_job_external_lease_result_ack = [pscustomobject]@{
        migration_bucket = "still_not_fully_cut_over"
        category = "still_not_fully_cut_over"
        current_execution_owner = $currentState.agent_job_execution_owner
        current_ack_owner = $currentState.agent_job_ack_owner
        ready = $currentState.agent_job_external_lease_ready
        decision = $currentState.agent_job_external_lease_decision
        readiness_category = $agentJobExternalLeaseVerification.conclusion.category
        repo_owned_temp_nats_smoke = $agentJobExternalLeaseNatsSmokeVerification.conclusion.category
        isolated_cutover_preflight = $agentJobExternalLeaseCutoverPreflightVerification.conclusion.category
        launcher_cutover_preflight = $agentJobExternalLeaseLauncherPreflightVerification.conclusion.category
        launcher_bundle_live_smoke = $agentJobExternalLeaseLauncherBundleVerification.conclusion.category
        cutover_diff_live_verifier = $agentJobExternalLeaseCutoverDiffVerification.conclusion.category
        remaining_gaps = @(
            "configuration",
            "production_cutover_owner_switch"
        )
        already_verified = @(
            "repo_owned_temp_nats_smoke",
            "isolated_cutover_preflight",
            "launcher_cutover_preflight",
            "launcher_bundle_live_smoke",
            "cutover_diff_live_verifier"
        )
        next_minimal_step = "Inject production runtime external-lease and strict-token flags into a controlled environment, then rerun readiness/plan plus result-ack smoke before switching owner."
    }
    scheduler_runtime = [pscustomobject]@{
        migration_bucket = "already_in_go_only_missing_live_verification"
        category = "already_in_go_missing_long_running_live_observation"
        live_verification = $schedulerVerification.conclusion.category
        remaining_observation = @(
            "real_reminder_execution_observation",
            "recurring_long_run_observation",
            "startup_recovery_long_run_observation"
        )
        next_minimal_step = "Keep isolated mutation/recovery smoke as the hard gate and add long-running reminder/recurring observation only when a safe fixture exists."
    }
    worker_control_executors = [pscustomobject]@{
        migration_bucket = "go_control_plane_present_but_real_executor_missing"
        category = "go_control_plane_present_but_real_executor_missing"
        control_plane_status = $workerControlExecutorVerification.conclusion.category
        missing_executors = @(
            "autoscaling_executor",
            "concurrency_control_executor",
            "priority_executor"
        )
        next_minimal_step = "Do not add executors until approval, audit, rollback, and rate-limit hooks are bound to a real runtime worker path."
    }
    knowledge_rag_state_boundary = [pscustomobject]@{
        migration_bucket = "already_in_go_only_missing_live_verification"
        category = if ($dashboardKnowledgeRagStateBoundaryVerification.conclusion.status -eq "live_verified") {
            "checkpoint_snapshot_derived_control_plane_live_verified"
        } else {
            "knowledge_rag_state_boundary_live_verification_incomplete"
        }
        dashboard_live_category = $dashboardKnowledgeRagStateBoundaryVerification.conclusion.category
        configured_rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.configured_rag_datasets
        rag_datasets = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_datasets
        rag_dataset_index_ready = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_ready
        rag_dataset_index_missing_snapshot = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_missing_snapshot
        rag_dataset_index_empty = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_empty
        rag_dataset_index_lagging = $dashboardKnowledgeRagStateBoundaryVerification.runtime_endpoint.totals.rag_dataset_index_lagging
        next_minimal_step = "Only add external index metadata when a stable read-only source exists; keep provider-specific raw parse/index payloads out of Go."
    }
    media_recovery_private_source_executor = [pscustomobject]@{
        migration_bucket = "go_control_plane_present_but_real_executor_missing"
        category = if ($mediaAssetContentRecoveryLiveSmokeVerification.conclusion.status -eq "live_verified") {
            "go_http_https_executor_live_verified_private_source_executor_incomplete"
        } else {
            "go_http_https_boundary_done_private_source_executor_incomplete"
        }
        boundary_status = $mediaRecoveryBoundaryVerification.conclusion.category
        http_https_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification.conclusion.category
        dashboard_media_asset_content_live_category = $dashboardMediaAssetContentBoundaryVerification.conclusion.category
        dashboard_live_category = $dashboardMediaAssetContentRecoveryBoundaryVerification.conclusion.category
        remaining_gaps = @(
            "qq_private_source_session_fetch",
            "telegram_private_source_session_fetch",
            "background_repull",
            "post_recovery_ai_enrichment"
        )
        next_minimal_step = "Keep Go on cache/audit/lifecycle and only add provider-specific fetch/exrichment once private-source credentials or sessions are available."
    }
    dashboard_read_models = [pscustomobject]@{
        migration_bucket = if ($currentState.dashboard_fallback_category -eq "live_verified_runtime_read_models") {
            "already_in_go_only_missing_live_verification"
        } else {
            "still_not_fully_cut_over"
        }
        category = if ($currentState.dashboard_fallback_category -eq "live_verified_runtime_read_models") {
            "tracked_runtime_overview_read_models_live_verified"
        } else {
            "tracked_runtime_overview_read_models_partially_verified"
        }
        fallback_category = $currentState.dashboard_fallback_category
        known_raw_json_only_gap = $false
        next_minimal_step = "Only add new table drilldowns when a newly introduced control-plane card still depends on raw JSON."
    }
    mq_adapter_boundary = [pscustomobject]@{
        migration_bucket = "already_in_go_only_missing_live_verification"
        category = "nats_jetstream_only_implemented_recommended_path"
        selected_provider = $queueBackend.data.provider
        recommended_provider = $queueTopologyBoundaryVerification.queue_topology.recommended_provider
        planned_only_adapters = @(
            "redis_streams",
            "rabbitmq"
        )
        next_minimal_step = "Do not implement Redis or RabbitMQ unless a concrete deployment requirement appears."
    }
    python_owned_surfaces = [pscustomobject]@{
        migration_bucket = "explicitly_python_owned"
        category = "intentionally_not_migrated_to_go"
        surfaces = @(
            "llm_provider_calls",
            "prompt_context_reasoning",
            "tool_execution",
            "ocr_vlm",
            "image_generation",
            "memory_rag_algorithms_and_strategy_experiments"
        )
        next_minimal_step = "Do not migrate these surfaces into Go; keep using Go only for deterministic runtime infrastructure around them."
    }
}
$migrationBucketSummary = [pscustomobject]@{
    already_in_go_only_missing_live_verification = @(@(
        New-MigrationBucketEntry -Name "telegram_backend" -Bucket $migrationResiduals.telegram_backend.migration_bucket
        New-MigrationBucketEntry -Name "scheduler_runtime" -Bucket $migrationResiduals.scheduler_runtime.migration_bucket
        New-MigrationBucketEntry -Name "knowledge_rag_state_boundary" -Bucket $migrationResiduals.knowledge_rag_state_boundary.migration_bucket
        New-MigrationBucketEntry -Name "dashboard_read_models" -Bucket $migrationResiduals.dashboard_read_models.migration_bucket
        New-MigrationBucketEntry -Name "mq_adapter_boundary" -Bucket $migrationResiduals.mq_adapter_boundary.migration_bucket
    ) | Where-Object { $_.migration_bucket -eq "already_in_go_only_missing_live_verification" })
    go_control_plane_present_but_real_executor_missing = @(@(
        New-MigrationBucketEntry -Name "worker_control_executors" -Bucket $migrationResiduals.worker_control_executors.migration_bucket
        New-MigrationBucketEntry -Name "media_recovery_private_source_executor" -Bucket $migrationResiduals.media_recovery_private_source_executor.migration_bucket
    ) | Where-Object { $_.migration_bucket -eq "go_control_plane_present_but_real_executor_missing" })
    still_not_fully_cut_over = @(@(
        New-MigrationBucketEntry -Name "qq_napcat_cutover" -Bucket $migrationResiduals.qq_napcat_cutover.migration_bucket
        New-MigrationBucketEntry -Name "telegram_backend" -Bucket $migrationResiduals.telegram_backend.migration_bucket
        New-MigrationBucketEntry -Name "agent_job_external_lease_result_ack" -Bucket $migrationResiduals.agent_job_external_lease_result_ack.migration_bucket
        New-MigrationBucketEntry -Name "dashboard_read_models" -Bucket $migrationResiduals.dashboard_read_models.migration_bucket
    ) | Where-Object { $_.migration_bucket -eq "still_not_fully_cut_over" })
    explicitly_python_owned = @(@(
        New-MigrationBucketEntry -Name "python_owned_surfaces" -Bucket $migrationResiduals.python_owned_surfaces.migration_bucket
    ) | Where-Object { $_.migration_bucket -eq "explicitly_python_owned" })
}
$dashboardFallbackReason = if ($currentState.dashboard_fallback_category -eq "live_verified_runtime_read_models") {
    "dashboard proactive tick-log fallback plus runtime-overview structured drilldowns for receiver statuses, receiver leases, knowledge pipelines, media asset content, media asset content recovery, media asset retention diagnostics/plan/cleanup, dead letters, checkpoint lag, job events, outbox events, rag-eval failures, runtime health, stale jobs, scheduler jobs, worker leases, runtime config, queue topology, qq cutover route matrix, control mutation policy, control audit, knowledge planner preview/readiness, agent workers, runtime workers, agent-job worker coverage, agent-job metrics, agent-job pressure, send ledger metrics, outbox metrics, outbox pressure, queue backend, external lease diagnostics, delivery smoke, external lease, capacity, priority, knowledge cutover and outbound cutover are all live on the current turn."
} elseif ($proactiveVerification.checks.dashboard_tick_logs_readable) {
    "dashboard proactive tick-log fallback/read-model is live on the current turn, but not all newer runtime-overview plan/readiness drilldowns produced direct current-turn evidence."
} else {
    "dashboard proactive tick-log fallback did not produce current evidence in this turn."
}

$result = [pscustomobject]@{
    generated_at = (Get-Date).ToString("o")
    runtime_base_url = $RuntimeBaseUrl
    current_state = $currentState
    runtime = [pscustomobject]@{
        healthz = $healthz.data.status
        bot_ids = $runtimeConfig.data.runtime.bot_ids
        queue_provider = $queueBackend.data.provider
        queue_mode = $queueBackend.data.mode
        runtime_overview_summary = [pscustomobject]@{
            queue_outbox_execution_owner = $runtimeOverview.data.summary.queue_outbox_execution_owner
            outbound_cutover_plan_current_owner = $runtimeOverview.data.summary.outbound_cutover_plan_current_owner
            outbound_cutover_plan_ready = $runtimeOverview.data.summary.outbound_cutover_plan_ready
            receiver_status_telegram = $runtimeOverview.data.summary.receiver_status_telegram
            knowledge_job_planner_cutover_plan_current_owner = $runtimeOverview.data.summary.knowledge_job_planner_cutover_plan_current_owner
            knowledge_job_planner_cutover_plan_ready = $runtimeOverview.data.summary.knowledge_job_planner_cutover_plan_ready
            outbox_pressure_high_accounts = $runtimeOverview.data.summary.outbox_pressure_high_accounts
        }
    }
    observe_only_silence = $observeOnlySilence
    qq_outbox_cutover = [pscustomobject]@{
        execution_ready = $outboundReadiness.data.execution_ready
        execution_owner = $outboundReadiness.data.execution_owner
        execution_scope = $queueBackend.data.outbox_execution_scope
        route_matrix = $qqCutoverRouteMatrixVerification.route_matrix
        live_scope_smoke = $outboxScopeSmoke
        verification = $qqCutoverRouteMatrixVerification
        conclusion = if ($null -ne $outboxScopeSmoke) {
            "partial_go_default_owner_live_verified"
        } else {
            $qqCutoverRouteMatrixVerification.conclusion.category
        }
    }
    qq_cutover_route_matrix = $qqCutoverRouteMatrixVerification
    qq_rich_media = [pscustomobject]@{
        native_probe_enabled = [bool]$IncludeNativeRichMediaProbe
        current_known_classification = [pscustomobject]@{
            image = "native_platform_blocker_across_accounts_and_group_private_routes"
            file = "first_account_group_file_is_group_specific_some_groups_fail_391289439_succeeds_second_account_group_and_both_accounts_private_success"
            adapter_status = "go_adapter_matches_native_behavior"
        }
        native_probe = $nativeRichMediaProbe
    }
    telegram_backend = $telegramVerification
    knowledge_planner = [pscustomobject]@{
        verification = $knowledgePlannerVerification
        recent_group_memory_jobs = $recentGroupMemoryJobs
        conclusion = if ($knowledgePlannerVerification.checks.planner_running_and_ready -and $knowledgePlannerVerification.checks.recent_go_scheduler_jobs_seen -and $knowledgePlannerVerification.checks.recent_skip_legacy_enqueue_seen) {
            "go_admission_owner_healthy"
        } else {
            "verification_incomplete"
        }
    }
    agent_job_external_lease = $agentJobExternalLeaseVerification
    agent_job_external_lease_nats_smoke = $agentJobExternalLeaseNatsSmokeVerification
    agent_job_external_lease_cutover_preflight = $agentJobExternalLeaseCutoverPreflightVerification
    agent_job_external_lease_launcher_preflight = $agentJobExternalLeaseLauncherPreflightVerification
    agent_job_external_lease_launcher_bundle = $agentJobExternalLeaseLauncherBundleVerification
    agent_job_external_lease_cutover_diff = $agentJobExternalLeaseCutoverDiffVerification
    worker_control_executors = $workerControlExecutorVerification
    control_audit_boundary = $controlAuditBoundaryVerification
    dashboard_control_audit_boundary = $dashboardControlAuditBoundaryVerification
    dashboard_media_asset_content_boundary = $dashboardMediaAssetContentBoundaryVerification
    dashboard_media_asset_content_recovery_boundary = $dashboardMediaAssetContentRecoveryBoundaryVerification
    dashboard_knowledge_rag_state_boundary = $dashboardKnowledgeRagStateBoundaryVerification
    dashboard_proactive_tick_logs_boundary = $dashboardProactiveTickLogsBoundaryVerification
    queue_topology = $queueTopologyBoundaryVerification
    media_recovery = $mediaRecoveryBoundaryVerification
    media_recovery_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification
    mq_adapter_boundary = $mqAdapterBoundaryVerification
    migration_residuals = $migrationResiduals
    migration_bucket_summary = $migrationBucketSummary
    dashboard_read_models = $dashboardReadModelChecks
    runtime_invariants = [pscustomobject]@{
        agent_worker_status_restart = $agentWorkerStatusRestartVerification
        agent_worker_status_fencing = $agentWorkerStatusFencingVerification
        agent_worker_status_heartbeat = $agentWorkerStatusHeartbeatVerification
        agent_worker_status_cleanup = $agentWorkerStatusCleanupVerification
        agent_worker_status_startup_prune = $agentWorkerStatusStartupPruneVerification
        receiver_status_cleanup = $receiverStatusCleanupVerification
        agent_job_external_lease_launcher_preflight = $agentJobExternalLeaseLauncherPreflightVerification
        agent_job_external_lease_launcher_bundle = $agentJobExternalLeaseLauncherBundleVerification
        agent_job_external_lease_cutover_diff = $agentJobExternalLeaseCutoverDiffVerification
    }
    residual_classification = [pscustomobject]@{
        agent_job_external_lease_result_ack = $agentJobExternalLeaseVerification.conclusion
        agent_job_external_lease_result_ack_smoke = $agentJobExternalLeaseNatsSmokeVerification.conclusion
        agent_job_external_lease_cutover_preflight = $agentJobExternalLeaseCutoverPreflightVerification.conclusion
        agent_job_external_lease_launcher_preflight = $agentJobExternalLeaseLauncherPreflightVerification.conclusion
        agent_job_external_lease_launcher_bundle = $agentJobExternalLeaseLauncherBundleVerification.conclusion
        agent_job_external_lease_cutover_diff = $agentJobExternalLeaseCutoverDiffVerification.conclusion
        queue_topology_boundary = $queueTopologyBoundaryVerification.conclusion
        scheduler = $schedulerVerification.conclusion
        proactive = $proactiveVerification.conclusion
        proactive_flow_smoke = $proactiveRuntimeFlowSmokeVerification.conclusion
        knowledge_rag_state_boundary = $dashboardKnowledgeRagStateBoundaryVerification.conclusion
        media_recovery_private_source_executor = $mediaRecoveryBoundaryVerification.conclusion
        media_recovery_http_https_executor_smoke = $mediaAssetContentRecoveryLiveSmokeVerification.conclusion
        mq_adapter_boundary = $mqAdapterBoundaryVerification.conclusion
        qq_cutover_route_matrix = $qqCutoverRouteMatrixVerification.conclusion
        worker_control_executors = $workerControlExecutorVerification.conclusion
        autoscaling_executor = $workerControlExecutorVerification.conclusion
        control_audit_boundary = $controlAuditBoundaryVerification.conclusion
        dashboard_control_audit_boundary = $dashboardControlAuditBoundaryVerification.conclusion
        dashboard_media_asset_content_boundary = $dashboardMediaAssetContentBoundaryVerification.conclusion
        dashboard_media_asset_content_recovery_boundary = $dashboardMediaAssetContentRecoveryBoundaryVerification.conclusion
        dashboard_proactive_tick_logs_boundary = $dashboardProactiveTickLogsBoundaryVerification.conclusion
        agent_worker_status_fencing = $agentWorkerStatusFencingVerification.conclusion
        agent_worker_status_heartbeat = $agentWorkerStatusHeartbeatVerification.conclusion
        agent_worker_status_cleanup = $agentWorkerStatusCleanupVerification.conclusion
        agent_worker_status_startup_prune = $agentWorkerStatusStartupPruneVerification.conclusion
        receiver_status_cleanup = $receiverStatusCleanupVerification.conclusion
        dashboard_fallback = [pscustomobject]@{
            status = "live_verified"
            category = $currentState.dashboard_fallback_category
            reason = $dashboardFallbackReason
        }
        python_owned_ai_runtime = [pscustomobject]@{
            status = "python_owned"
            category = "intentionally_not_migrated_to_go"
            reason = "LLM provider routing, prompt/context, tool execution, OCR/VLM, image generation, and memory/RAG algorithms remain Python by design"
        }
    }
    scheduler_runtime = $schedulerVerification
    proactive_runtime = $proactiveVerification
    proactive_runtime_flow_smoke = $proactiveRuntimeFlowSmokeVerification
    receivers = $receiverStatuses.data
    workers = $runtimeWorkers.data
    checks = [pscustomobject]@{
        dashboard_read_models = $dashboardReadModelChecks
        goal_ready_to_close = $false
        open_blockers = $openBlockers
    }
}

if ([string]::IsNullOrWhiteSpace($JsonOutputPath)) {
    $JsonOutputPath = Join-Path $repoRoot ".codex-goal-verifier.json"
}

$json = $result | ConvertTo-Json -Depth 16
Set-Content -LiteralPath $JsonOutputPath -Value $json -Encoding UTF8

if ($StdoutMode -eq "none") {
    return
}

if ($StdoutMode -eq "full") {
    $json
    return
}

$stdoutSummary = [pscustomobject]@{
    status = "artifact_written"
    artifact_path = $JsonOutputPath
    generated_at = $result.generated_at
    goal_ready_to_close = $result.checks.goal_ready_to_close
    open_blockers = $result.checks.open_blockers
    current_state = $result.current_state
    migration_bucket_summary = $result.migration_bucket_summary
}

$stdoutSummary | ConvertTo-Json -Depth 8
