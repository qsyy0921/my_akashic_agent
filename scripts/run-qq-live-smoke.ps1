param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$PrimaryBotId = "1049511700",
    [string]$SecondaryBotId = "2365524513",
    [string]$PrimaryChannel = "qq_1049511700",
    [string]$SecondaryChannel = "qq_2365524513",
    [string]$LogPath = "",
    [string]$PeerTriggerPrefix = "",
    [int]$LogWaitSeconds = 10,
    [switch]$SkipMarkSucceeded,
    [switch]$SkipLogCheck
)

$ErrorActionPreference = "Stop"

function Get-RepoRoot {
    return Split-Path -Parent $PSScriptRoot
}

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

function Wait-LogPattern {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [int]$TimeoutSeconds = 10
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return $false
    }

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $match = Select-String -Path $Path -Pattern $Pattern -SimpleMatch -ErrorAction SilentlyContinue
        if ($match) {
            return $true
        }
        Start-Sleep -Milliseconds 800
    }
    return $false
}

function New-SmokeMessage {
    param(
        [Parameter(Mandatory = $true)][string]$FromBotId,
        [Parameter(Mandatory = $true)][string]$ToBotId,
        [string]$Prefix = ""
    )

    $token = "GO_QQ_LIVE_SMOKE_{0}_{1}_{2}_{3}" -f (Get-Date -Format "yyyyMMddTHHmmss"), $FromBotId, $ToBotId, ([guid]::NewGuid().ToString("N").Substring(0, 8))
    if ([string]::IsNullOrWhiteSpace($Prefix)) {
        return @{
            token = $token
            content = $token
        }
    }

    return @{
        token = $token
        content = ("{0} {1}" -f $Prefix.Trim(), $token)
    }
}

function Invoke-PrivateTextSmoke {
    param(
        [Parameter(Mandatory = $true)][string]$FromBotId,
        [Parameter(Mandatory = $true)][string]$ToBotId,
        [Parameter(Mandatory = $true)][string]$ChannelAlias,
        [Parameter(Mandatory = $true)][hashtable]$ChannelByAccount,
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$Content,
        [Parameter(Mandatory = $true)][string]$Token,
        [string]$LogPath = "",
        [int]$LogWaitSeconds = 10,
        [switch]$SkipMarkSucceeded,
        [switch]$SkipLogCheck
    )

    $eventId = "qq-live-smoke:{0}:{1}:{2}" -f $FromBotId, $ToBotId, ([guid]::NewGuid().ToString("N"))
    $outboundBody = @{
        event_id = $eventId
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $FromBotId
            conversation_id = $ToBotId
            conversation_type = "private"
        }
        content = $Content
        metadata = @{
            source = "qq_live_smoke"
            live_smoke = "true"
            max_attempts = "1"
        }
    }

    $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbound" -Body $outboundBody

    $readiness = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/readiness" -Body @{
        event_id = $eventId
        channel_by_account = $ChannelByAccount
    }

    if (-not $readiness.data.ready) {
        throw "delivery dispatch readiness blocked for ${eventId}: $($readiness.data.reason)"
    }

    $dispatch = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/send" -Body @{
        event_id = $eventId
        channel_by_account = $ChannelByAccount
    }

    if (-not $SkipMarkSucceeded) {
        $encodedEventId = [uri]::EscapeDataString($eventId)
        $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbox/$encodedEventId/succeeded" -Body @{}
    }

    $encodedStateId = [uri]::EscapeDataString($eventId)
    $outboxState = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/outbox/$encodedStateId"

    $logObserved = $false
    if (-not $SkipLogCheck -and -not [string]::IsNullOrWhiteSpace($LogPath)) {
        $logObserved = Wait-LogPattern -Path $LogPath -Pattern $Token -TimeoutSeconds $LogWaitSeconds
    }

    return [pscustomobject]@{
        event_id = $eventId
        from_bot_id = $FromBotId
        to_bot_id = $ToBotId
        channel = $ChannelAlias
        token = $Token
        readiness = $readiness.data
        dispatch = $dispatch.data
        outbox = $outboxState.data
        log_observed = $logObserved
    }
}

$repoRoot = Get-RepoRoot
if ([string]::IsNullOrWhiteSpace($LogPath)) {
    $LogPath = Join-Path $repoRoot "logs\akashic-local-run.err.log"
}

$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$adapterHealth = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/delivery-adapters/health?timeout_seconds=5"

$channelByAccount = @{
    $PrimaryBotId = $PrimaryChannel
    $SecondaryBotId = $SecondaryChannel
}

$primaryMessage = New-SmokeMessage -FromBotId $PrimaryBotId -ToBotId $SecondaryBotId -Prefix $PeerTriggerPrefix
$secondaryMessage = New-SmokeMessage -FromBotId $SecondaryBotId -ToBotId $PrimaryBotId -Prefix $PeerTriggerPrefix

$results = @()
$results += Invoke-PrivateTextSmoke -FromBotId $PrimaryBotId -ToBotId $SecondaryBotId -ChannelAlias $PrimaryChannel -ChannelByAccount $channelByAccount -BaseUrl $RuntimeBaseUrl -Content $primaryMessage.content -Token $primaryMessage.token -LogPath $LogPath -LogWaitSeconds $LogWaitSeconds -SkipMarkSucceeded:$SkipMarkSucceeded -SkipLogCheck:$SkipLogCheck
$results += Invoke-PrivateTextSmoke -FromBotId $SecondaryBotId -ToBotId $PrimaryBotId -ChannelAlias $SecondaryChannel -ChannelByAccount $channelByAccount -BaseUrl $RuntimeBaseUrl -Content $secondaryMessage.content -Token $secondaryMessage.token -LogPath $LogPath -LogWaitSeconds $LogWaitSeconds -SkipMarkSucceeded:$SkipMarkSucceeded -SkipLogCheck:$SkipLogCheck

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    runtime_config = $runtimeConfig.data
    delivery_adapter_health = $adapterHealth.data
    log_path = $LogPath
    peer_trigger_prefix = $PeerTriggerPrefix
    cases = $results
} | ConvertTo-Json -Depth 12
