param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$LogPath = "",
    [int]$PortOwner = 8765
)

$ErrorActionPreference = "Stop"

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Uri
    )

    Invoke-RestMethod -Method $Method -Uri $Uri -TimeoutSec 30
}

function Get-DefaultLogPath {
    $repoRoot = Split-Path -Parent $PSScriptRoot
    $botLog = Join-Path $repoRoot "logs\bot.log"
    $legacyLog = Join-Path $repoRoot "logs\akashic-local-run.err.log"
    if (Test-Path $botLog) {
        return $botLog
    }
    return $legacyLog
}

function Get-InstancePid {
    param([string]$InstanceId)

    if ([string]::IsNullOrWhiteSpace($InstanceId)) {
        return $null
    }
    $parts = $InstanceId -split ":"
    if ($parts.Count -lt 3) {
        return $null
    }
    $pidText = $parts[$parts.Count - 2]
    $parsedPid = 0
    if (-not [int]::TryParse($pidText, [ref]$parsedPid)) {
        return $null
    }
    return $parsedPid
}

if ([string]::IsNullOrWhiteSpace($LogPath)) {
    $LogPath = Get-DefaultLogPath
}

$workers = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/agent-worker-statuses"
$activeWorkers = @($workers.data.workers | Where-Object {
    $_.worker_id -in @(
        "akashic-python-worker:image",
        "akashic-python-worker:knowledge",
        "akashic-python-worker:outbox"
    )
})

$portOwnerPid = $null
$portOwnerConnection = Get-NetTCPConnection -LocalPort $PortOwner -ErrorAction SilentlyContinue |
    Where-Object { $_.State -eq "Listen" } |
    Select-Object -First 1
if ($portOwnerConnection) {
    $portOwnerPid = [int]$portOwnerConnection.OwningProcess
}

$conflictLines = @()
$reportFailedLines = @()
if (Test-Path $LogPath) {
    $conflictLines = @(Select-String -Path $LogPath -Pattern "agent worker status lease conflict" | ForEach-Object { $_.Line })
    $reportFailedLines = @(Select-String -Path $LogPath -Pattern "agent worker status report failed" | ForEach-Object { $_.Line })
}

$instancePidChecks = @($activeWorkers | ForEach-Object {
    $instancePid = Get-InstancePid -InstanceId $_.instance_id
    [pscustomobject]@{
        worker_id = $_.worker_id
        instance_id = $_.instance_id
        instance_pid = $instancePid
        matches_port_owner = ($null -ne $instancePid -and $null -ne $portOwnerPid -and $instancePid -eq $portOwnerPid)
        lease_active = $_.lease_active
        status = $_.status
    }
})

[pscustomobject]@{
    generated_at = (Get-Date).ToString("o")
    log_path = $LogPath
    port_owner = [pscustomobject]@{
        port = $PortOwner
        pid = $portOwnerPid
    }
    workers = $instancePidChecks
    log_summary = [pscustomobject]@{
        conflict_lines = $conflictLines
        report_failed_lines = $reportFailedLines
    }
    checks = [pscustomobject]@{
        active_worker_records_present = ($instancePidChecks.Count -ge 3)
        active_worker_instance_ids_match_current_port_owner = (
            $instancePidChecks.Count -ge 3 -and
            @($instancePidChecks | Where-Object { -not $_.matches_port_owner }).Count -eq 0
        )
        no_worker_status_conflict_lines = ($conflictLines.Count -eq 0)
        no_worker_status_report_failed_lines = ($reportFailedLines.Count -eq 0)
    }
    conclusion = [pscustomobject]@{
        status = if (
            $instancePidChecks.Count -ge 3 -and
            @($instancePidChecks | Where-Object { -not $_.matches_port_owner }).Count -eq 0 -and
            $conflictLines.Count -eq 0 -and
            $reportFailedLines.Count -eq 0
        ) { "live_verified" } else { "verification_failed" }
        category = if (
            $instancePidChecks.Count -ge 3 -and
            @($instancePidChecks | Where-Object { -not $_.matches_port_owner }).Count -eq 0 -and
            $conflictLines.Count -eq 0 -and
            $reportFailedLines.Count -eq 0
        ) { "controlled_takeover_verified_after_restart" } else { "restart_conflict_or_runtime_gap_detected" }
        reason = if (
            $instancePidChecks.Count -ge 3 -and
            @($instancePidChecks | Where-Object { -not $_.matches_port_owner }).Count -eq 0 -and
            $conflictLines.Count -eq 0 -and
            $reportFailedLines.Count -eq 0
        ) {
            "After restart, active Python worker-status instance ids were replaced with the current main process pid and the fresh log contains no lease conflict or worker-status report failure lines."
        } else {
            "Current runtime still shows stale worker instance ownership or fresh log conflict lines after restart."
        }
    }
} | ConvertTo-Json -Depth 8
