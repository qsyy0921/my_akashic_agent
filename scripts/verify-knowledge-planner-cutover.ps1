param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$AgentRuntimeLogPath = "",
    [string]$PythonLogPath = "",
    [int]$RecentPlannerBuckets = 3,
    [int]$RecentJobs = 12,
    [int]$TailLines = 400
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

function Get-RecentPlannerBuckets {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [int]$Tail = 400,
        [int]$Limit = 3
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return @()
    }

    $lines = Get-Content -LiteralPath $Path -Tail $Tail
    $matches = foreach ($line in $lines) {
        if ($line -match '^(?<ts>\S+\s+\S+)\s+knowledge job planner bucket=(?<bucket>\d+)\s+targets=(?<targets>\d+)\s+created_or_existing=(?<created>\d+)\s+suppressed=(?<suppressed>\d+)\s+group_memory=(?<group>\d+)\s+rag_ingest=(?<rag>\d+)') {
            [pscustomobject]@{
                timestamp = $matches.ts
                bucket = [int64]$matches.bucket
                targets = [int]$matches.targets
                created_or_existing = [int]$matches.created
                suppressed = [int]$matches.suppressed
                group_memory = [int]$matches.group
                rag_ingest = [int]$matches.rag
            }
        }
    }

    return @($matches | Sort-Object bucket -Descending | Select-Object -First $Limit)
}

function Get-RecentPythonKnowledgeSignals {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [int]$Tail = 400
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return [pscustomobject]@{
            skip_legacy_enqueue = @()
            processed_jobs = @()
        }
    }

    $lines = Get-Content -LiteralPath $Path -Tail $Tail
    $skip = foreach ($line in $lines) {
        if ($line -match '^\S+\s+\S+.*\[agent_runtime_knowledge_worker\] skip legacy enqueue') {
            [pscustomobject]@{
                line = $line.Trim()
            }
        }
    }
    $processed = foreach ($line in $lines) {
        if ($line -match "^\S+\s+\S+.*\[agent_runtime_knowledge_worker\] processed .*'job_id': '([^']+)'") {
            [pscustomobject]@{
                line = $line.Trim()
                job_id = $matches[1]
            }
        }
    }

    [pscustomobject]@{
        skip_legacy_enqueue = @($skip | Select-Object -Last 12)
        processed_jobs = @($processed | Select-Object -Last 24)
    }
}

function Summarize-RecentJobs {
    param(
        [Parameter(Mandatory = $true)]$Jobs
    )

    return @($Jobs | ForEach-Object {
        [pscustomobject]@{
            job_id = $_.job_id
            status = $_.status
            scheduler = $_.metadata.scheduler
            conversation_id = $_.route.conversation_id
            updated_at = $_.updated_at
        }
    })
}

if ([string]::IsNullOrWhiteSpace($AgentRuntimeLogPath)) {
    $AgentRuntimeLogPath = Join-Path (Join-Path (Split-Path -Parent $PSScriptRoot) "logs") "agent-runtime-local.err.log"
}
if ([string]::IsNullOrWhiteSpace($PythonLogPath)) {
    $PythonLogPath = Join-Path (Join-Path (Split-Path -Parent $PSScriptRoot) "logs") "akashic-local-run.err.log"
}

$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$runtimeWorkers = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-workers"
$readiness = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/knowledge-job-planner/readiness"
$cutoverPlan = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/knowledge-job-planner/cutover-plan"
$jobs = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/jobs?type=group_memory_extract&limit=$RecentJobs"

$plannerBuckets = Get-RecentPlannerBuckets -Path $AgentRuntimeLogPath -Tail $TailLines -Limit $RecentPlannerBuckets
$pythonSignals = Get-RecentPythonKnowledgeSignals -Path $PythonLogPath -Tail $TailLines
$jobSummaries = Summarize-RecentJobs -Jobs $jobs.data

$latestSchedulerOk = $jobSummaries.Count -gt 0 -and @($jobSummaries | Where-Object { $_.scheduler -eq "agent-runtime-knowledge-job-planner" }).Count -gt 0
$recentSucceeded = @($jobSummaries | Where-Object { $_.status -eq "succeeded" -and $_.scheduler -eq "agent-runtime-knowledge-job-planner" }).Count
$skipSeen = @($pythonSignals.skip_legacy_enqueue).Count -gt 0
$plannerRunning = ($readiness.data.ready -eq $true -and $cutoverPlan.data.current_admission_owner -eq "go_runtime_knowledge_job_planner")

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    runtime = [pscustomobject]@{
        knowledge_job_planner_enabled = $runtimeConfig.data.workers.knowledge_job_planner_enabled
        planner_worker_running = @($runtimeWorkers.data.workers | Where-Object { $_.name -eq "knowledge_job_planner" } | Select-Object -First 1).running
    }
    readiness = $readiness.data
    cutover_plan = [pscustomobject]@{
        ready = $cutoverPlan.data.ready
        decision = $cutoverPlan.data.decision
        current_admission_owner = $cutoverPlan.data.current_admission_owner
        desired_admission_owner = $cutoverPlan.data.desired_admission_owner
    }
    recent_planner_buckets = $plannerBuckets
    recent_group_memory_jobs = $jobSummaries
    python_signals = [pscustomobject]@{
        skip_legacy_enqueue = $pythonSignals.skip_legacy_enqueue
        processed_jobs = $pythonSignals.processed_jobs
    }
    checks = [pscustomobject]@{
        planner_running_and_ready = $plannerRunning
        recent_planner_buckets_seen = (@($plannerBuckets).Count -ge [Math]::Min($RecentPlannerBuckets, 1))
        recent_go_scheduler_jobs_seen = $latestSchedulerOk
        recent_go_scheduler_job_successes = $recentSucceeded
        recent_skip_legacy_enqueue_seen = $skipSeen
    }
} | ConvertTo-Json -Depth 12
