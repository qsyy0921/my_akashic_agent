param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$DashboardBaseUrl = "http://127.0.0.1:2236"
)

$ErrorActionPreference = "Stop"

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Uri
    )

    Invoke-RestMethod -Method $Method -Uri $Uri -TimeoutSec 30
}

$queueBackend = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-backend"
$queueTopology = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-topology"
$runtimeOverview = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-overview"
$dashboardOverview = Invoke-JsonRequest -Method "GET" -Uri "$DashboardBaseUrl/api/dashboard/runtime-overview"

$providerCapabilities = @($queueBackend.data.provider_capabilities)
$dashboardQueueBackend = $dashboardOverview.queue_backend
$dashboardProviderCapabilities = @($dashboardQueueBackend.provider_capabilities)

$natsCapability = $providerCapabilities | Where-Object { $_.provider -eq "nats_jetstream" } | Select-Object -First 1
$redisCapability = $providerCapabilities | Where-Object { $_.provider -eq "redis_streams" } | Select-Object -First 1
$rabbitCapability = $providerCapabilities | Where-Object { $_.provider -eq "rabbitmq" } | Select-Object -First 1
$selectedCapability = $queueBackend.data.selected_provider_capability
$dashboardSelectedCapability = $dashboardQueueBackend.selected_provider_capability

$checks = [pscustomobject]@{
    selected_provider_visible = (-not [string]::IsNullOrWhiteSpace($queueBackend.data.provider))
    nats_is_recommended_first_backend = (
        $queueBackend.data.recommended_first_backend -eq "nats_jetstream" -and
        $null -ne $natsCapability -and
        $natsCapability.recommended -eq $true -and
        $natsCapability.implemented -eq $true -and
        $natsCapability.supports_external_lease -eq $true -and
        $natsCapability.supports_agent_job_result_ack -eq $true
    )
    redis_and_rabbit_are_planned_only = (
        $null -ne $redisCapability -and
        $null -ne $rabbitCapability -and
        $redisCapability.implemented -eq $false -and
        $rabbitCapability.implemented -eq $false -and
        $redisCapability.status -eq "planned" -and
        $rabbitCapability.status -eq "planned"
    )
    queue_topology_recommended_provider_matches = (
        $queueTopology.data.recommended_provider -eq "nats_jetstream"
    )
    runtime_overview_summary_keeps_queue_provider = (
        $runtimeOverview.data.summary.queue_provider_status -ne $null -and
        $dashboardOverview.summary.queue_backend_provider -eq $queueBackend.data.provider
    )
    dashboard_preserves_provider_capabilities = (
        $dashboardProviderCapabilities.Count -ge 4 -and
        $dashboardSelectedCapability.provider -eq $queueBackend.data.provider -and
        @($dashboardProviderCapabilities | Where-Object { $_.provider -eq "nats_jetstream" }).Count -eq 1
    )
}

$allChecksPassed = (
    $checks.selected_provider_visible -and
    $checks.nats_is_recommended_first_backend -and
    $checks.redis_and_rabbit_are_planned_only -and
    $checks.queue_topology_recommended_provider_matches -and
    $checks.runtime_overview_summary_keeps_queue_provider -and
    $checks.dashboard_preserves_provider_capabilities
)

[pscustomobject]@{
    generated_at = (Get-Date).ToString("o")
    runtime_base_url = $RuntimeBaseUrl
    dashboard_base_url = $DashboardBaseUrl
    queue_backend = [pscustomobject]@{
        provider = $queueBackend.data.provider
        mode = $queueBackend.data.mode
        recommended_first_backend = $queueBackend.data.recommended_first_backend
        selected_provider_capability = $selectedCapability
        provider_capabilities = $providerCapabilities
    }
    queue_topology = [pscustomobject]@{
        provider = $queueTopology.data.provider
        recommended_provider = $queueTopology.data.recommended_provider
        selected_provider = $queueTopology.data.selected_provider
        work_kinds = $queueTopology.data.work_kinds
    }
    dashboard_runtime_overview = [pscustomobject]@{
        summary = [pscustomobject]@{
            queue_backend_provider = $dashboardOverview.summary.queue_backend_provider
            queue_backend_mode = $dashboardOverview.summary.queue_backend_mode
        }
        queue_backend = [pscustomobject]@{
            provider = $dashboardQueueBackend.provider
            mode = $dashboardQueueBackend.mode
            supported_providers = $dashboardQueueBackend.supported_providers
            selected_provider_capability = $dashboardSelectedCapability
            provider_capabilities = $dashboardProviderCapabilities
        }
    }
    checks = $checks
    conclusion = [pscustomobject]@{
        status = if ($allChecksPassed) { "live_verified" } else { "verification_failed" }
        category = if ($allChecksPassed) {
            "mq_adapter_boundary_live_verified_with_dashboard_read_model"
        } else {
            "mq_adapter_boundary_read_model_or_runtime_gap_detected"
        }
        reason = if ($allChecksPassed) {
            "NATS JetStream remains the only implemented recommended external MQ path, Redis Streams and RabbitMQ remain planned-only boundaries, and dashboard runtime-overview preserves the capability matrix without owning adapter logic."
        } else {
            "At least one MQ adapter boundary check failed or the dashboard queue-backend read model dropped provider capability details."
        }
    }
} | ConvertTo-Json -Depth 12
