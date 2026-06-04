param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$DashboardBaseUrl = "http://127.0.0.1:2236",
    [int]$DiagnosticsLimit = 120,
    [int]$PlanSampleLimit = 40,
    [string]$SampleAssetID = "",
    [string]$OperatorID = "goal-verifier"
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

function Get-UrlScheme {
    param([string]$Url)

    if ([string]::IsNullOrWhiteSpace($Url)) {
        return "(empty)"
    }

    try {
        $uri = [uri]$Url
        if ([string]::IsNullOrWhiteSpace($uri.Scheme)) {
            return "(none)"
        }
        return $uri.Scheme.ToLowerInvariant()
    } catch {
        if ($Url -match '^[A-Za-z]:\\') {
            return "windows_path"
        }
        return "invalid"
    }
}

function Group-CountValues {
    param(
        [Parameter(Mandatory = $true)]$Values,
        [int]$Top = 0
    )

    $groups = @(@($Values) | Group-Object | Sort-Object Count -Descending)
    if ($Top -gt 0) {
        $groups = @($groups | Select-Object -First $Top)
    }
    return @($groups | ForEach-Object {
        @{
            name = [string]$_.Name
            count = $_.Count
        }
    })
}

$runtimeOverview = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-overview"
$contentDiagnostics = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/media-assets/content-diagnostics?limit=$DiagnosticsLimit"
$mediaAssets = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/media-assets?limit=$DiagnosticsLimit"
$dashboardRuntimeOverview = Invoke-JsonRequest -Method "GET" -Uri "$DashboardBaseUrl/api/dashboard/runtime-overview"

$assets = @($mediaAssets.data)
$diagnosticItems = @($contentDiagnostics.data.items)

$schemeCounts = Group-CountValues -Values ($assets | ForEach-Object { Get-UrlScheme $_.url })
$kindCounts = Group-CountValues -Values ($assets | ForEach-Object { $_.kind })
$conversationCounts = Group-CountValues -Values ($assets | ForEach-Object { $_.channel.conversation_type })

$selectedAsset = $null
if (-not [string]::IsNullOrWhiteSpace($SampleAssetID)) {
    $selectedAsset = @($assets | Where-Object { $_.asset_id -eq $SampleAssetID } | Select-Object -First 1)
}
if ($null -eq $selectedAsset) {
    $selectedAsset = @($assets | Select-Object -First 1)
}

$sampleAssetID = if ($null -ne $selectedAsset) { $selectedAsset.asset_id } else { "" }
$sampleAccessPlan = $null
$sampleRecoveryPlan = $null
$samplePreflight = $null
$dashboardSampleRecoveryPlan = $null

if (-not [string]::IsNullOrWhiteSpace($sampleAssetID)) {
    $encodedAssetID = [uri]::EscapeDataString($sampleAssetID)
    $sampleAccessPlan = (Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/media-assets/content-access-plan?asset_id=$encodedAssetID").data
    $sampleRecoveryPlan = (Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/media-assets/content-recovery-plan?asset_id=$encodedAssetID").data
    $samplePreflight = (Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/media-assets/content-recovery/preflight?asset_id=$encodedAssetID&operator_id=$([uri]::EscapeDataString($OperatorID))").data
    $dashboardSampleRecoveryPlan = Invoke-JsonRequest -Method "GET" -Uri "$DashboardBaseUrl/api/dashboard/media-assets/content-recovery-plan?asset_id=$encodedAssetID"
}

$planSamples = New-Object System.Collections.Generic.List[object]
foreach ($asset in @($assets | Select-Object -First $PlanSampleLimit)) {
    $plan = (Invoke-JsonRequest -Method "GET" -Uri ($RuntimeBaseUrl + "/v1/media-assets/content-recovery-plan?asset_id=" + [uri]::EscapeDataString($asset.asset_id))).data
    $planSamples.Add([ordered]@{
        asset_id = $asset.asset_id
        scheme = Get-UrlScheme $asset.url
        kind = $asset.kind
        account_id = $asset.channel.account_id
        conversation_type = $asset.channel.conversation_type
        reason = $plan.reason
        future_executor_scope = $plan.future_executor_scope
    })
}

$planReasonCounts = Group-CountValues -Values ($planSamples | ForEach-Object { $_.reason })
$planScopeCounts = Group-CountValues -Values ($planSamples | ForEach-Object { if ([string]::IsNullOrWhiteSpace($_.future_executor_scope)) { "(empty)" } else { $_.future_executor_scope } })
$planSchemeReasonCounts = Group-CountValues -Values ($planSamples | ForEach-Object { "{0}|{1}|{2}" -f $_.scheme, $_.reason, $(if ([string]::IsNullOrWhiteSpace($_.future_executor_scope)) { "(empty)" } else { $_.future_executor_scope }) })
$diagnosticReasonCounts = Group-CountValues -Values ($diagnosticItems | ForEach-Object { $_.content_reason }) -Top 10
$diagnosticStatusCounts = Group-CountValues -Values ($diagnosticItems | ForEach-Object { $_.content_status }) -Top 10

$runtimeMediaRecoveryDetail = $runtimeOverview.data.media_asset_content_recovery
if ($null -eq $runtimeMediaRecoveryDetail -and $null -ne $runtimeOverview.data.details) {
    $runtimeMediaRecoveryDetail = $runtimeOverview.data.details.media_asset_content_recovery
}

$dashboardHasMediaRecoveryDetail = $false
if ($null -ne $dashboardRuntimeOverview.media_asset_content_recovery) {
    $dashboardHasMediaRecoveryDetail = $true
}
if (-not $dashboardHasMediaRecoveryDetail -and $null -ne $dashboardRuntimeOverview.details -and $null -ne $dashboardRuntimeOverview.details.media_asset_content_recovery) {
    $dashboardHasMediaRecoveryDetail = $true
}

$dashboardHasMediaRecoveryCard = [bool]@($dashboardRuntimeOverview.cards | Where-Object { $_.id -eq "media_asset_content_recovery" } | Select-Object -First 1)
$hasHttpLikeAssets = [bool]@($assets | Where-Object { (Get-UrlScheme $_.url) -in @("http", "https") } | Select-Object -First 1)
$hasFileAssets = [bool]@($assets | Where-Object { (Get-UrlScheme $_.url) -eq "file" } | Select-Object -First 1)
$allDiagnosticsForbidden = ($contentDiagnostics.data.totals.assets -gt 0 -and $contentDiagnostics.data.totals.forbidden -eq $contentDiagnostics.data.totals.assets)
$dominantOperatorConfigBoundary = [bool]@($planSamples | Where-Object {
    $_.scheme -eq "file" -and
    $_.reason -eq "media_asset_content_recovery_fix_content_roots" -and
    $_.future_executor_scope -eq "operator_runtime_config"
} | Select-Object -First 1)

$category = "verification_incomplete"
if ($null -ne $runtimeMediaRecoveryDetail -and $null -ne $sampleRecoveryPlan -and $null -ne $dashboardSampleRecoveryPlan) {
    if ($hasHttpLikeAssets) {
        $category = "go_control_plane_live_http_executor_candidates_present"
    } elseif ($allDiagnosticsForbidden -and $hasFileAssets -and $dominantOperatorConfigBoundary) {
        $category = "go_control_plane_live_operator_runtime_config_boundary"
    } else {
        $category = "go_control_plane_live_mixed_boundary"
    }
}

$conclusionStatus = if ($category -eq "verification_incomplete") { "verification_incomplete" } else { "live_verified" }
$conclusionReason = if ($category -eq "go_control_plane_live_http_executor_candidates_present") {
    "Runtime exposes live media diagnostics, recovery planning, preflight, and runtime-overview recovery detail; this sample set includes HTTP/HTTPS assets, so Go cache-recovery executor candidates exist in the current registry."
} elseif ($category -eq "go_control_plane_live_operator_runtime_config_boundary") {
    "Runtime exposes live media diagnostics, recovery planning, preflight, and dashboard proxy paths; current sampled assets are file:// uploads outside configured content roots, so the active boundary is operator runtime config rather than provider-specific private-source fetching."
} elseif ($category -eq "go_control_plane_live_mixed_boundary") {
    "Runtime exposes live media diagnostics, recovery planning, and preflight, but sampled assets span mixed schemes or reasons and need manual review before declaring a single dominant recovery boundary."
} else {
    "At least one required media recovery endpoint or read-model surface did not produce current-turn evidence."
}
$planSamplesCount = [int]$planSamples.Count
$dashboardRecoveryPlanProxyReady = ($null -ne $dashboardSampleRecoveryPlan)
$contentRecoveryPlanEndpointReachable = ($null -ne $sampleRecoveryPlan)
$contentRecoveryPreflightEndpointReachable = ($null -ne $samplePreflight)
$runtimeOverviewHasMediaRecoveryDetail = ($null -ne $runtimeMediaRecoveryDetail)
$dashboardRuntimeOverviewMissingMediaRecoveryDetail = (-not $dashboardHasMediaRecoveryDetail)
$samplePreflightRequiresApproval = ($null -ne $samplePreflight -and $samplePreflight.reason -eq "missing_approval_id")

$runtimeOverviewPayload = @{}
$runtimeOverviewPayload["media_asset_content_ready"] = $runtimeOverview.data.summary.media_asset_content_ready
$runtimeOverviewPayload["media_asset_content_reason"] = $runtimeOverview.data.summary.media_asset_content_reason
$runtimeOverviewPayload["media_asset_content_recovery_ready"] = $runtimeOverview.data.summary.media_asset_content_recovery_ready
$runtimeOverviewPayload["media_asset_content_recovery_reason"] = $runtimeOverview.data.summary.media_asset_content_recovery_reason
$runtimeOverviewPayload["media_asset_content_recovery_applied"] = $runtimeOverview.data.summary.media_asset_content_recovery_applied
$runtimeOverviewPayload["media_asset_content_recovery_failed"] = $runtimeOverview.data.summary.media_asset_content_recovery_failed
$runtimeOverviewPayload["media_asset_retention_ready"] = $runtimeOverview.data.summary.media_asset_retention_ready
$runtimeOverviewPayload["media_asset_retention_reason"] = $runtimeOverview.data.summary.media_asset_retention_reason
$runtimeOverviewPayload["media_asset_content_recovery_detail"] = $runtimeMediaRecoveryDetail

$diagnosticsPayload = @{}
$diagnosticsPayload["totals"] = $contentDiagnostics.data.totals
$diagnosticsPayload["scheme_counts"] = $schemeCounts
$diagnosticsPayload["kind_counts"] = $kindCounts
$diagnosticsPayload["conversation_type_counts"] = $conversationCounts
$diagnosticsPayload["top_reason_counts"] = $diagnosticReasonCounts
$diagnosticsPayload["top_status_counts"] = $diagnosticStatusCounts

$planSamplesPayload = @{}
$planSamplesPayload["sample_count"] = $planSamplesCount
$planSamplesPayload["top_reason_counts"] = $planReasonCounts
$planSamplesPayload["top_future_scope_counts"] = $planScopeCounts
$planSamplesPayload["top_scheme_reason_scope_counts"] = $planSchemeReasonCounts

$sampleAssetPayload = @{}
$sampleAssetPayload["asset"] = $selectedAsset
$sampleAssetPayload["access_plan"] = $sampleAccessPlan
$sampleAssetPayload["recovery_plan"] = $sampleRecoveryPlan
$sampleAssetPayload["preflight"] = $samplePreflight
$sampleAssetPayload["dashboard_recovery_plan"] = $dashboardSampleRecoveryPlan

$dashboardPayload = @{}
$dashboardPayload["runtime_overview_has_media_content_recovery_card"] = $dashboardHasMediaRecoveryCard
$dashboardPayload["runtime_overview_has_media_content_recovery_detail"] = $dashboardHasMediaRecoveryDetail
$dashboardPayload["dashboard_recovery_plan_proxy_ready"] = $dashboardRecoveryPlanProxyReady

$checksPayload = @{}
$checksPayload["content_diagnostics_endpoint_reachable"] = $true
$checksPayload["content_recovery_plan_endpoint_reachable"] = $contentRecoveryPlanEndpointReachable
$checksPayload["content_recovery_preflight_endpoint_reachable"] = $contentRecoveryPreflightEndpointReachable
$checksPayload["runtime_overview_has_media_recovery_detail"] = $runtimeOverviewHasMediaRecoveryDetail
$checksPayload["dashboard_runtime_overview_missing_media_recovery_detail"] = $dashboardRuntimeOverviewMissingMediaRecoveryDetail
$checksPayload["dashboard_recovery_plan_proxy_reachable"] = $dashboardRecoveryPlanProxyReady
$checksPayload["has_http_https_assets"] = $hasHttpLikeAssets
$checksPayload["has_file_assets"] = $hasFileAssets
$checksPayload["all_sampled_assets_forbidden"] = $allDiagnosticsForbidden
$checksPayload["operator_runtime_config_boundary_dominant"] = $dominantOperatorConfigBoundary
$checksPayload["sample_preflight_requires_approval"] = $samplePreflightRequiresApproval

$conclusionPayload = @{}
$conclusionPayload["status"] = $conclusionStatus
$conclusionPayload["category"] = $category
$conclusionPayload["reason"] = $conclusionReason

$result = @{}
$result["runtime_base_url"] = $RuntimeBaseUrl
$result["dashboard_base_url"] = $DashboardBaseUrl
$result["runtime_overview"] = $runtimeOverviewPayload
$result["diagnostics"] = $diagnosticsPayload
$result["plan_samples"] = $planSamplesPayload
$result["sample_asset"] = $sampleAssetPayload
$result["dashboard"] = $dashboardPayload
$result["checks"] = $checksPayload
$result["conclusion"] = $conclusionPayload

$result | ConvertTo-Json -Depth 16
