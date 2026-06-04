param(
    [switch]$Foreground,
    [switch]$EnableOutboxDeliveryWorker,
    [ValidateSet("", "true", "false")][string]$QQGroupSendEnabled = "",
    [string]$RuntimeAddr = "",
    [string]$RuntimeStateDir = "",
    [string]$ShadowAuditPath = "",
    [string]$LogPrefix = "",
    [string]$OutboxDeliveryAllowedKinds = "",
    [string]$OutboxDeliveryAllowedKindsByAccount = "",
    [string]$OutboxDeliveryAllowedKindsByAccountConversationType = "",
    [string]$OutboxDeliveryAllowedKindsByAccountConversationId = "",
    [string]$QueueBackend = "",
    [string]$QueueDSN = "",
    [string]$QueueMode = "",
    [string]$QueueConsumerConcurrency = "",
    [string]$QueueMaxInFlight = "",
    [ValidateSet("", "true", "false")][string]$QueueExternalLeaseCutover = "",
    [ValidateSet("", "true", "false")][string]$QueueDualReadSmokePassed = "",
    [ValidateSet("", "true", "false")][string]$QueueStateLeaseWorkersDisabled = "",
    [ValidateSet("", "true", "false")][string]$QueueExternalLeaseAgentJobEnabled = "",
    [ValidateSet("", "true", "false")][string]$QueueAgentJobDuplicateSmokePassed = "",
    [ValidateSet("", "true", "false")][string]$QueueAgentJobFlowSmokePassed = "",
    [ValidateSet("", "true", "false")][string]$AgentJobStrictLeaseToken = "",
    [switch]$EnableKnowledgeJobPlanner,
    [int]$KnowledgeJobPlannerIntervalSeconds = 300,
    [int]$KnowledgeJobPlannerMaxAttempts = 3,
    [int]$KnowledgeJobPlannerRagMaxMessages = 200,
    [switch]$KnowledgeJobPlannerRunOnStart,
    [int]$StartupTimeoutSeconds = 60
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir
$serviceDir = Join-Path $repoRoot "services\agent-runtime"
$logDir = Join-Path $repoRoot "logs"

if (-not (Test-Path $serviceDir)) {
    throw "agent-runtime service directory not found: $serviceDir"
}

New-Item -ItemType Directory -Force -Path $logDir | Out-Null

$goExe = (Get-Command go -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -First 1)
if (-not $goExe) {
    $candidate = Join-Path $env:LOCALAPPDATA "Programs\Go\bin\go.exe"
    if (Test-Path $candidate) {
        $goExe = $candidate
    }
}
if (-not $goExe) {
    throw "go.exe not found in PATH or $env:LOCALAPPDATA\Programs\Go\bin\go.exe"
}

function Resolve-LauncherSetting {
    param(
        [string]$ExplicitValue,
        [string]$EnvironmentVariable,
        [string]$DefaultValue = ""
    )

    if (-not [string]::IsNullOrWhiteSpace($ExplicitValue)) {
        return $ExplicitValue
    }
    $envValue = [Environment]::GetEnvironmentVariable($EnvironmentVariable)
    if (-not [string]::IsNullOrWhiteSpace($envValue)) {
        return $envValue
    }
    return $DefaultValue
}

function Add-LauncherEnvValue {
    param(
        [hashtable]$EnvBlock,
        [string]$Key,
        [string]$ExplicitValue,
        [string]$EnvironmentVariable
    )

    $value = Resolve-LauncherSetting -ExplicitValue $ExplicitValue -EnvironmentVariable $EnvironmentVariable
    if (-not [string]::IsNullOrWhiteSpace($value)) {
        $EnvBlock[$Key] = $value
    }
}

function Get-LauncherHealthConfig {
    param(
        [Parameter(Mandatory = $true)][string]$Address
    )

    $match = [regex]::Match($Address, ":(\d+)$")
    if (-not $match.Success) {
        throw "RuntimeAddr must end with a TCP port: $Address"
    }

    $port = [int]$match.Groups[1].Value
    $hostPart = $Address.Substring(0, $Address.Length - $match.Groups[1].Value.Length - 1).Trim()
    if ([string]::IsNullOrWhiteSpace($hostPart) -or $hostPart -eq "0.0.0.0" -or $hostPart -eq "::") {
        $healthHost = "127.0.0.1"
    } else {
        $healthHost = $hostPart
    }

    return [pscustomobject]@{
        Port = $port
        HealthUrl = "http://{0}:{1}/healthz" -f $healthHost, $port
    }
}

$outboxWorkerEnabled = if ($EnableOutboxDeliveryWorker) {
    "true"
} elseif ($env:AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED) {
    $env:AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED
} else {
    "false"
}
$qqGroupSendEnabled = if (-not [string]::IsNullOrWhiteSpace($QQGroupSendEnabled)) {
    $QQGroupSendEnabled
} elseif ($env:AKASHIC_QQ_GROUP_SEND_ENABLED) {
    $env:AKASHIC_QQ_GROUP_SEND_ENABLED
} else {
    "false"
}
$outboxAllowedKinds = if ($env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS) {
    $env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS
} elseif (-not [string]::IsNullOrWhiteSpace($OutboxDeliveryAllowedKinds)) {
    $OutboxDeliveryAllowedKinds
} else {
    ""
}
$outboxAllowedKindsByAccount = if ($env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT) {
    $env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT
} elseif (-not [string]::IsNullOrWhiteSpace($OutboxDeliveryAllowedKindsByAccount)) {
    $OutboxDeliveryAllowedKindsByAccount
} else {
    ""
}
$outboxAllowedKindsByAccountConversationType = if ($env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE) {
    $env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE
} elseif (-not [string]::IsNullOrWhiteSpace($OutboxDeliveryAllowedKindsByAccountConversationType)) {
    $OutboxDeliveryAllowedKindsByAccountConversationType
} else {
    ""
}
$outboxAllowedKindsByAccountConversationId = if ($env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_ID) {
    $env:AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_ID
} elseif (-not [string]::IsNullOrWhiteSpace($OutboxDeliveryAllowedKindsByAccountConversationId)) {
    $OutboxDeliveryAllowedKindsByAccountConversationId
} else {
    ""
}

$plannerIntervalSeconds = if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS) {
    $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS
} else {
    [string]$KnowledgeJobPlannerIntervalSeconds
}
$plannerMaxAttempts = if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS) {
    $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS
} else {
    [string]$KnowledgeJobPlannerMaxAttempts
}
$plannerRagMaxMessages = if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES) {
    $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES
} else {
    [string]$KnowledgeJobPlannerRagMaxMessages
}
$plannerRunOnStart = if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START) {
    $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START
} elseif ($KnowledgeJobPlannerRunOnStart) {
    "true"
} else {
    "true"
}

$resolvedRuntimeAddr = Resolve-LauncherSetting -ExplicitValue $RuntimeAddr -EnvironmentVariable "AKASHIC_RUNTIME_ADDR" -DefaultValue "127.0.0.1:8780"
$resolvedRuntimeStateDir = Resolve-LauncherSetting -ExplicitValue $RuntimeStateDir -EnvironmentVariable "AKASHIC_RUNTIME_STATE_DIR" -DefaultValue (Join-Path $repoRoot ".akashic-workspace\agent-runtime")
$resolvedShadowAuditPath = Resolve-LauncherSetting -ExplicitValue $ShadowAuditPath -EnvironmentVariable "AKASHIC_SHADOW_AUDIT_PATH" -DefaultValue (Join-Path $repoRoot ".akashic-workspace\shadow\runtime-audit.jsonl")
$healthConfig = Get-LauncherHealthConfig -Address $resolvedRuntimeAddr
$resolvedLogPrefix = if (-not [string]::IsNullOrWhiteSpace($LogPrefix)) { $LogPrefix } else { "agent-runtime-local" }
$stdoutLog = Join-Path $logDir ("{0}.out.log" -f $resolvedLogPrefix)
$stderrLog = Join-Path $logDir ("{0}.err.log" -f $resolvedLogPrefix)

$listener = Get-NetTCPConnection -State Listen -LocalPort $healthConfig.Port -ErrorAction SilentlyContinue | Select-Object -First 1
if ($listener) {
    throw "port $($healthConfig.Port) is already in use by pid $($listener.OwningProcess)"
}

$envBlock = @{
    "AKASHIC_RUNTIME_ADDR" = $resolvedRuntimeAddr
    "AKASHIC_RUNTIME_STATE_DIR" = $resolvedRuntimeStateDir
    "AKASHIC_SHADOW_AUDIT_PATH" = $resolvedShadowAuditPath
    "AKASHIC_BOT_IDS" = "1049511700,2365524513"
    "AKASHIC_ONEBOT_WS_URLS" = "qq=ws://127.0.0.1:3001,qq_1049511700=ws://127.0.0.1:3001,qq_2365524513=ws://127.0.0.1:3002"
    "AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT" = "1049511700=qq_1049511700,2365524513=qq_2365524513"
    "AKASHIC_ONEBOT_ACCESS_TOKEN" = "NcatBot"
    "AKASHIC_QQ_GROUP_SEND_ENABLED" = $qqGroupSendEnabled
    "AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED" = $outboxWorkerEnabled
}
if (-not [string]::IsNullOrWhiteSpace($outboxAllowedKinds)) {
    $envBlock["AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS"] = $outboxAllowedKinds
}
if (-not [string]::IsNullOrWhiteSpace($outboxAllowedKindsByAccount)) {
    $envBlock["AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT"] = $outboxAllowedKindsByAccount
}
if (-not [string]::IsNullOrWhiteSpace($outboxAllowedKindsByAccountConversationType)) {
    $envBlock["AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_TYPE"] = $outboxAllowedKindsByAccountConversationType
}
if (-not [string]::IsNullOrWhiteSpace($outboxAllowedKindsByAccountConversationId)) {
    $envBlock["AKASHIC_OUTBOX_DELIVERY_ALLOWED_KINDS_BY_ACCOUNT_CONVERSATION_ID"] = $outboxAllowedKindsByAccountConversationId
}
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_BACKEND" -ExplicitValue $QueueBackend -EnvironmentVariable "AKASHIC_QUEUE_BACKEND"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_DSN" -ExplicitValue $QueueDSN -EnvironmentVariable "AKASHIC_QUEUE_DSN"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_MODE" -ExplicitValue $QueueMode -EnvironmentVariable "AKASHIC_QUEUE_MODE"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_CONSUMER_CONCURRENCY" -ExplicitValue $QueueConsumerConcurrency -EnvironmentVariable "AKASHIC_QUEUE_CONSUMER_CONCURRENCY"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_MAX_IN_FLIGHT" -ExplicitValue $QueueMaxInFlight -EnvironmentVariable "AKASHIC_QUEUE_MAX_IN_FLIGHT"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER" -ExplicitValue $QueueExternalLeaseCutover -EnvironmentVariable "AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED" -ExplicitValue $QueueDualReadSmokePassed -EnvironmentVariable "AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED" -ExplicitValue $QueueStateLeaseWorkersDisabled -EnvironmentVariable "AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED" -ExplicitValue $QueueExternalLeaseAgentJobEnabled -EnvironmentVariable "AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED" -ExplicitValue $QueueAgentJobDuplicateSmokePassed -EnvironmentVariable "AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED" -ExplicitValue $QueueAgentJobFlowSmokePassed -EnvironmentVariable "AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED"
Add-LauncherEnvValue -EnvBlock $envBlock -Key "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN" -ExplicitValue $AgentJobStrictLeaseToken -EnvironmentVariable "AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN"

$plannerEnabled = $EnableKnowledgeJobPlanner -or ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED -eq "true")
if ($plannerEnabled) {
    $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_ENABLED"] = "true"
    $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_INTERVAL_SECONDS"] = $plannerIntervalSeconds
    $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_MAX_ATTEMPTS"] = $plannerMaxAttempts
    $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_RAG_MAX_MESSAGES"] = $plannerRagMaxMessages
    $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_RUN_ON_START"] = $plannerRunOnStart
    if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID) {
        $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID"] = $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_AGENT_ID
    }
    if ($env:AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID) {
        $envBlock["AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID"] = $env:AKASHIC_KNOWLEDGE_JOB_PLANNER_WORKER_ID
    }
}

if ($env:TELEGRAM_BOT_TOKEN) {
    $envBlock["TELEGRAM_BOT_TOKEN"] = $env:TELEGRAM_BOT_TOKEN
}
if ($env:AKASHIC_TELEGRAM_BOT_TOKEN) {
    $envBlock["AKASHIC_TELEGRAM_BOT_TOKEN"] = $env:AKASHIC_TELEGRAM_BOT_TOKEN
}

$envAssignments = ($envBlock.GetEnumerator() | Sort-Object Name | ForEach-Object {
    '$env:{0}=''{1}''' -f $_.Key, ($_.Value -replace "'", "''")
}) -join "; "
$launchCommand = "$envAssignments; & '$goExe' run ./cmd/agent-runtime"

if ($Foreground) {
    Push-Location $serviceDir
    try {
        Invoke-Expression $launchCommand
    }
    finally {
        Pop-Location
    }
    exit 0
}

$process = Start-Process -FilePath "powershell.exe" `
    -ArgumentList @("-NoProfile", "-Command", $launchCommand) `
    -WorkingDirectory $serviceDir `
    -RedirectStandardOutput $stdoutLog `
    -RedirectStandardError $stderrLog `
    -WindowStyle Hidden `
    -PassThru

$deadline = (Get-Date).AddSeconds($StartupTimeoutSeconds)
$healthUrl = $healthConfig.HealthUrl
$ready = $false
while ((Get-Date) -lt $deadline) {
    if ($process.HasExited) {
        break
    }
    try {
        $response = Invoke-WebRequest -Uri $healthUrl -UseBasicParsing -TimeoutSec 2
        if ($response.StatusCode -eq 200) {
            $ready = $true
            break
        }
    }
    catch {
        Start-Sleep -Milliseconds 500
    }
}

if (-not $ready) {
    if (-not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
    }
    $tail = ""
    if (Test-Path $stderrLog) {
        $tail = (Get-Content $stderrLog -Tail 20) -join [Environment]::NewLine
    }
    throw "agent-runtime did not become healthy on $healthUrl within $StartupTimeoutSeconds seconds.`n$tail"
}

[pscustomobject]@{
    pid = $process.Id
    go_exe = $goExe
    runtime_addr = $resolvedRuntimeAddr
    runtime_state_dir = $resolvedRuntimeStateDir
    shadow_audit_path = $resolvedShadowAuditPath
    health_url = $healthUrl
    stdout_log = $stdoutLog
    stderr_log = $stderrLog
}
