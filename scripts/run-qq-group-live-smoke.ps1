param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$AccountId = "1049511700",
    [string]$ChannelAlias = "qq_1049511700",
    [string]$GroupId = "",
    [string]$ImageCaption = "GO_QQ_GROUP_IMAGE_SMOKE",
    [string]$FileCaption = "GO_QQ_GROUP_FILE_SMOKE",
    [switch]$SkipMarkSucceeded
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

function Resolve-GroupId {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$AccountId,
        [string]$PreferredGroupId = ""
    )

    $targets = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/observe-targets"
    $items = @($targets.data.targets | Where-Object {
        $_.enabled -eq $true -and
        $_.channel.kind -eq "qq" -and
        $_.channel.account_id -eq $AccountId -and
        $_.channel.conversation_type -eq "group"
    })
    if ([string]::IsNullOrWhiteSpace($PreferredGroupId)) {
        if ($items.Count -eq 0) {
            throw "no enabled QQ observe-only group target found for account $AccountId"
        }
        return [pscustomobject]@{
            group_id = [string]$items[0].channel.conversation_id
            target = $items[0]
            targets = $items
        }
    }

    $matched = $items | Where-Object { [string]$_.channel.conversation_id -eq $PreferredGroupId } | Select-Object -First 1
    if ($null -eq $matched) {
        throw "preferred group id $PreferredGroupId was not found in enabled observe targets for account $AccountId"
    }
    return [pscustomobject]@{
        group_id = [string]$matched.channel.conversation_id
        target = $matched
        targets = $items
    }
}

function New-SmokeAssets {
    $tempDir = Join-Path $env:TEMP ("akashic-go-qq-group-smoke-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $tempDir | Out-Null

    $imagePath = Join-Path $tempDir "smoke.png"
    $filePath = Join-Path $tempDir "smoke.txt"

    [IO.File]::WriteAllBytes(
        $imagePath,
        [Convert]::FromBase64String("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO2p8L8AAAAASUVORK5CYII=")
    )
    Set-Content -LiteralPath $filePath -Value "go qq group file smoke" -NoNewline -Encoding utf8

    return [pscustomobject]@{
        temp_dir = $tempDir
        image_path = $imagePath
        file_path = $filePath
    }
}

function Invoke-GroupSmokeCase {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$AccountId,
        [Parameter(Mandatory = $true)][string]$ChannelAlias,
        [Parameter(Mandatory = $true)][string]$GroupId,
        [Parameter(Mandatory = $true)][hashtable]$ChannelByAccount,
        [Parameter(Mandatory = $true)][hashtable]$Case,
        [switch]$SkipMarkSucceeded
    )

    $eventId = "qq-group-live-smoke:{0}:{1}" -f $Case.kind, ([guid]::NewGuid().ToString("N"))
    $body = @{
        event_id = $eventId
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $AccountId
            conversation_id = $GroupId
            conversation_type = "group"
        }
        content = $Case.content
        metadata = @{
            source = "qq_group_live_smoke"
            live_smoke = "true"
            max_attempts = "1"
        }
    }
    if ($Case.attachments.Count -gt 0) {
        $body["attachments"] = $Case.attachments
    }

    $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbound" -Body $body
    $readiness = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/readiness" -Body @{
        event_id = $eventId
        channel_by_account = $ChannelByAccount
    }

    $dispatch = $null
    $dispatchError = $null
    try {
        $dispatch = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/send" -Body @{
            event_id = $eventId
            channel_by_account = $ChannelByAccount
        }
        if (-not $SkipMarkSucceeded) {
            $null = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbox/$([uri]::EscapeDataString($eventId))/succeeded" -Body @{}
        }
    }
    catch {
        $dispatchError = $_.ErrorDetails.Message
        if ([string]::IsNullOrWhiteSpace($dispatchError)) {
            $dispatchError = $_.Exception.Message
        }
    }

    $state = Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/outbox/$([uri]::EscapeDataString($eventId))"
    return [pscustomobject]@{
        kind = $Case.kind
        event_id = $eventId
        account_id = $AccountId
        group_id = $GroupId
        channel = $ChannelAlias
        readiness = $readiness.data
        dispatch = if ($dispatch) { $dispatch.data } else { $null }
        error = $dispatchError
        outbox = $state.data
    }
}

$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$adapterHealth = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/delivery-adapters/health?timeout_seconds=5"
$resolvedGroup = Resolve-GroupId -BaseUrl $RuntimeBaseUrl -AccountId $AccountId -PreferredGroupId $GroupId
$assets = New-SmokeAssets

$channelByAccount = @{
    "1049511700" = "qq_1049511700"
    "2365524513" = "qq_2365524513"
}
$channelByAccount[$AccountId] = $ChannelAlias

$textToken = "GO_QQ_GROUP_TEXT_SMOKE_{0}_{1}" -f (Get-Date -Format "yyyyMMddTHHmmss"), ([guid]::NewGuid().ToString("N").Substring(0, 8))
$cases = @(
    @{
        kind = "text"
        content = $textToken
        attachments = @()
    },
    @{
        kind = "image"
        content = $ImageCaption
        attachments = @(
            @{
                kind = "image"
                url = $assets.image_path
                name = "smoke.png"
                mime_type = "image/png"
            }
        )
    },
    @{
        kind = "file"
        content = $FileCaption
        attachments = @(
            @{
                kind = "file"
                url = $assets.file_path
                name = "smoke.txt"
                mime_type = "text/plain"
            }
        )
    }
)

$results = foreach ($case in $cases) {
    Invoke-GroupSmokeCase `
        -BaseUrl $RuntimeBaseUrl `
        -AccountId $AccountId `
        -ChannelAlias $ChannelAlias `
        -GroupId $resolvedGroup.group_id `
        -ChannelByAccount $channelByAccount `
        -Case $case `
        -SkipMarkSucceeded:$SkipMarkSucceeded
}

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    runtime_config = $runtimeConfig.data
    delivery_adapter_health = $adapterHealth.data
    observe_target = $resolvedGroup.target
    available_group_targets = @($resolvedGroup.targets | ForEach-Object { $_.channel.conversation_id })
    temp_assets = $assets
    cases = $results
} | ConvertTo-Json -Depth 12
