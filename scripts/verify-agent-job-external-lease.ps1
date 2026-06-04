param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$DesiredExecutionOwner = "python_ai_worker_with_nats_result_ack"
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

function Get-EnvPresenceMap {
    param($RuntimeConfig)

    $map = @{}
    foreach ($entry in @($RuntimeConfig.data.environment)) {
        $map[$entry.key] = [bool]$entry.present
    }
    return $map
}

function Get-QueueTopologyAgentJobKind {
    param($QueueTopology)

    return @($QueueTopology.data.work_kinds | Where-Object { $_.work_kind -eq "agent_job" } | Select-Object -First 1)
}

$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$readiness = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-external-lease/readiness"
$plan = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-job-external-lease/plan?desired_execution_owner=$([uri]::EscapeDataString($DesiredExecutionOwner))"
$queueBackend = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-backend"
$queueTopology = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-topology"
$runtimeOverview = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-overview"
$agentWorkers = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-worker-statuses"

$envPresence = Get-EnvPresenceMap -RuntimeConfig $runtimeConfig
$agentJobKind = Get-QueueTopologyAgentJobKind -QueueTopology $queueTopology
$selectedProviderCapability = @($queueBackend.data.provider_capabilities | Where-Object { $_.status -eq "selected" } | Select-Object -First 1)

$smokeFlagKeys = @(
    "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED",
    "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED"
)
$baseCutoverKeys = @(
    "AKASHIC_QUEUE_BACKEND",
    "AKASHIC_QUEUE_DSN",
    "AKASHIC_QUEUE_MODE",
    "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER",
    "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED",
    "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED"
)
$resultAckKeys = @(
    "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN",
    "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"
)

$missingSmokeFlags = @($smokeFlagKeys | Where-Object { -not $envPresence.ContainsKey($_) -or -not $envPresence[$_] })
$missingBaseCutover = @($baseCutoverKeys | Where-Object { -not $envPresence.ContainsKey($_) -or -not $envPresence[$_] })
$missingResultAckFlags = @($resultAckKeys | Where-Object { -not $envPresence.ContainsKey($_) -or -not $envPresence[$_] })

$configurationBlockers = New-Object System.Collections.Generic.List[string]
$ownerBlockers = New-Object System.Collections.Generic.List[string]
$smokeBlockers = New-Object System.Collections.Generic.List[string]
$workerCoverageBlockers = New-Object System.Collections.Generic.List[string]

foreach ($blocker in @($readiness.data.blockers)) {
    switch ($blocker) {
        "strict_lease_token_disabled" { $configurationBlockers.Add($blocker); continue }
        "queue_external_lease_agent_job_flag_disabled" { $configurationBlockers.Add($blocker); continue }
        "external_lease_not_configured" { $configurationBlockers.Add($blocker); continue }
        "external_lease_execution_not_allowed" { $configurationBlockers.Add($blocker); continue }
        "agent_job_not_allowed_in_external_lease" { $ownerBlockers.Add($blocker); continue }
        "agent_job_result_ack_owner_not_enabled" { $ownerBlockers.Add($blocker); continue }
        default { }
    }
}

if (-not [bool]$readiness.data.agent_job_worker_ready) {
    $workerCoverageBlockers.Add("agent_job_worker_not_ready")
}
if ($missingSmokeFlags.Count -gt 0) {
    $smokeBlockers.Add("agent_job_duplicate_or_flow_smoke_flags_missing")
}

$currentScope = if ($queueBackend.data.agent_job_execution_owner -eq "python_ai_worker_with_nats_result_ack") {
    "nats_result_ack"
} else {
    "state_store_lease"
}

$conclusionCategory = if ([bool]$readiness.data.ready) {
    "ready_for_nats_result_ack_cutover"
} elseif ($configurationBlockers.Count -gt 0 -or $missingBaseCutover.Count -gt 0 -or $missingResultAckFlags.Count -gt 0) {
    "blocked_by_configuration_and_cutover_flags"
} elseif ($ownerBlockers.Count -gt 0) {
    "blocked_by_execution_owner_scope"
} elseif ($smokeBlockers.Count -gt 0) {
    "blocked_by_missing_smoke_evidence"
} elseif ($workerCoverageBlockers.Count -gt 0) {
    "blocked_by_worker_coverage"
} else {
    "blocked_by_multiple_runtime_gates"
}

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    desired_execution_owner = $DesiredExecutionOwner
    readiness = $readiness.data
    plan = [pscustomobject]@{
        ready = $plan.data.ready
        decision = $plan.data.decision
        desired_execution_owner = $plan.data.desired_execution_owner
        recommended_execution_owner = $plan.data.recommended_execution_owner
        current_execution_owner = $plan.data.current_execution_owner
        blockers = $plan.data.blockers
        required_checks = $plan.data.required_checks
        enable_steps = $plan.data.enable_steps
        verification_steps = $plan.data.verification_steps
        rollback_steps = $plan.data.rollback_steps
    }
    runtime_flags = [pscustomobject]@{
        agent_job_strict_lease_token = [bool]$runtimeConfig.data.workers.agent_job_strict_lease_token
        queue_external_lease_agent_job_enabled = [bool]$runtimeConfig.data.workers.queue_external_lease_agent_job_enabled
        env_presence = [pscustomobject]@{
            base_cutover_flags = [ordered]@{
                AKASHIC_QUEUE_BACKEND = [bool]$envPresence["AKASHIC_QUEUE_BACKEND"]
                AKASHIC_QUEUE_DSN = [bool]$envPresence["AKASHIC_QUEUE_DSN"]
                AKASHIC_QUEUE_MODE = [bool]$envPresence["AKASHIC_QUEUE_MODE"]
                AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER = [bool]$envPresence["AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER"]
                AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED = [bool]$envPresence["AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED"]
                AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED = [bool]$envPresence["AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED"]
            }
            result_ack_flags = [ordered]@{
                AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN = [bool]$envPresence["AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"]
                AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED = [bool]$envPresence["AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"]
            }
            smoke_flags = [ordered]@{
                AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED = [bool]$envPresence["AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED"]
                AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED = [bool]$envPresence["AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED"]
            }
        }
        missing_base_cutover_flags = $missingBaseCutover
        missing_result_ack_flags = $missingResultAckFlags
        missing_smoke_flags = $missingSmokeFlags
    }
    queue_backend = [pscustomobject]@{
        provider = $queueBackend.data.provider
        mode = $queueBackend.data.mode
        external_queue_configured = [bool]$queueBackend.data.external_queue_configured
        external_queue_active = [bool]$queueBackend.data.external_queue_active
        agent_job_execution_owner = $queueBackend.data.agent_job_execution_owner
        selected_provider_capability = $selectedProviderCapability
    }
    queue_topology = [pscustomobject]@{
        agent_job = $agentJobKind
    }
    runtime_overview = [pscustomobject]@{
        agent_job_external_lease_ready = $runtimeOverview.data.summary.agent_job_external_lease_ready
        agent_job_external_lease_reason = $runtimeOverview.data.summary.agent_job_external_lease_reason
        agent_job_external_lease_blockers = $runtimeOverview.data.summary.agent_job_external_lease_blockers
        agent_job_external_lease_result_ack_ready = $runtimeOverview.data.summary.agent_job_external_lease_result_ack_ready
        agent_job_external_lease_execution_owner = $runtimeOverview.data.summary.agent_job_external_lease_execution_owner
        queue_provider_supports_external_lease = $runtimeOverview.data.summary.queue_provider_supports_external_lease
        queue_provider_supports_agent_job_result_ack = $runtimeOverview.data.summary.queue_provider_supports_agent_job_result_ack
        queue_topology_agent_job_execution_owner = $runtimeOverview.data.summary.queue_topology_agent_job_execution_owner
        queue_topology_agent_job_ack_owner = $runtimeOverview.data.summary.queue_topology_agent_job_ack_owner
    }
    agent_workers = $agentWorkers.data
    blocker_buckets = [pscustomobject]@{
        configuration = @($configurationBlockers | Select-Object -Unique)
        execution_owner = @($ownerBlockers | Select-Object -Unique)
        smoke = @($smokeBlockers | Select-Object -Unique)
        worker_coverage = @($workerCoverageBlockers | Select-Object -Unique)
    }
    checks = [pscustomobject]@{
        readiness_endpoint_reachable = $true
        plan_endpoint_reachable = $true
        queue_backend_endpoint_reachable = $true
        queue_topology_endpoint_reachable = $true
        runtime_overview_summary_visible = $true
        worker_coverage_ready = [bool]$readiness.data.agent_job_worker_ready
        strict_lease_token_enabled = [bool]$runtimeConfig.data.workers.agent_job_strict_lease_token
        result_ack_scope_flag_enabled = [bool]$runtimeConfig.data.workers.queue_external_lease_agent_job_enabled
        nats_provider_selected = ($queueBackend.data.provider -eq "nats_jetstream")
        selected_provider_supports_agent_job_result_ack = [bool]$selectedProviderCapability.supports_agent_job_result_ack
        current_owner_is_state_store = ($queueBackend.data.agent_job_execution_owner -eq "python_ai_worker_state_store_lease")
        queue_topology_matches_current_owner = ($null -ne $agentJobKind -and $agentJobKind.execution_owner -eq $queueBackend.data.agent_job_execution_owner)
    }
    conclusion = [pscustomobject]@{
        status = "live_verified"
        category = $conclusionCategory
        current_scope = $currentScope
        reason = if ([bool]$readiness.data.ready) {
            "agent_job result-ack cutover gates are satisfied in the current runtime."
        } elseif ($conclusionCategory -eq "blocked_by_configuration_and_cutover_flags") {
            "Current runtime is still on local state-store ownership and is missing external-lease base flags and/or result-ack flags."
        } elseif ($conclusionCategory -eq "blocked_by_execution_owner_scope") {
            "External lease ownership has not expanded from state-store agent_job execution to Python-with-NATS-result-ack scope."
        } elseif ($conclusionCategory -eq "blocked_by_missing_smoke_evidence") {
            "Current runtime still lacks explicit duplicate/flow smoke evidence for safe agent_job result-ack cutover."
        } elseif ($conclusionCategory -eq "blocked_by_worker_coverage") {
            "Python worker coverage is not ready enough for agent_job result-ack cutover."
        } else {
            "Multiple current-turn gates still block agent_job result-ack cutover."
        }
    }
} | ConvertTo-Json -Depth 16
