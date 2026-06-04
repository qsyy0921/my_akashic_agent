param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$FirstAccountId = "1049511700",
    [string]$SecondAccountId = "2365524513",
    [string]$FirstAccountPrivatePeerId = "2365524513",
    [string]$FirstAccountAllowedGroupId = "391289439",
    [string]$FirstAccountGroupId = "3219982",
    [string]$SecondAccountGroupId = "284331268",
    [int]$PollSeconds = 12
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

function New-SmokeAssets {
    $tempDir = Join-Path $env:TEMP ("akashic-go-outbox-scope-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $tempDir | Out-Null

    $filePath = Join-Path $tempDir "scope-smoke.txt"
    $imagePath = Join-Path $tempDir "scope-smoke.png"

    Set-Content -LiteralPath $filePath -Value ("go outbox scope smoke {0}" -f (Get-Date).ToString("s")) -NoNewline -Encoding utf8
    [IO.File]::WriteAllBytes(
        $imagePath,
        [Convert]::FromBase64String("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO2p8L8AAAAASUVORK5CYII=")
    )

    return [pscustomobject]@{
        temp_dir = $tempDir
        file_path = $filePath
        image_path = $imagePath
    }
}

function Submit-OutboxEvent {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$EventId,
        [Parameter(Mandatory = $true)][hashtable]$Channel,
        [Parameter(Mandatory = $true)][string]$Content,
        [array]$Attachments = @(),
        [hashtable]$Metadata = @{}
    )

    $body = @{
        event_id = $EventId
        channel = $Channel
        content = $Content
        metadata = $Metadata
    }
    if ($Attachments.Count -gt 0) {
        $body["attachments"] = $Attachments
    }

    return Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/outbound" -Body $body
}

function Get-OutboxState {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$EventId
    )

    $encoded = [uri]::EscapeDataString($EventId)
    return (Invoke-JsonRequest -Method "GET" -Uri "$BaseUrl/v1/outbox/$encoded").data
}

function Wait-ForOutboxState {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$EventId,
        [Parameter(Mandatory = $true)][scriptblock]$Condition,
        [int]$TimeoutSeconds = 12
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $history = New-Object System.Collections.Generic.List[object]
    do {
        $state = Get-OutboxState -BaseUrl $BaseUrl -EventId $EventId
        $history.Add([pscustomobject]@{
            timestamp = (Get-Date).ToString("o")
            status = $state.status
            attempts = $state.attempts
            leased_by = $state.leased_by
        })
        if (& $Condition $state) {
            return [pscustomobject]@{
                state = $state
                history = $history.ToArray()
                condition_met = $true
            }
        }
        Start-Sleep -Milliseconds 900
    } while ((Get-Date) -lt $deadline)

    $finalState = Get-OutboxState -BaseUrl $BaseUrl -EventId $EventId
    $history.Add([pscustomobject]@{
        timestamp = (Get-Date).ToString("o")
        status = $finalState.status
        attempts = $finalState.attempts
        leased_by = $finalState.leased_by
    })
    return [pscustomobject]@{
        state = $finalState
        history = $history.ToArray()
        condition_met = $false
    }
}

function Test-DispatchReadinessBlocked {
    param(
        [Parameter(Mandatory = $true)][string]$BaseUrl,
        [Parameter(Mandatory = $true)][string]$EventId,
        [Parameter(Mandatory = $true)][string]$ExpectedMessage
    )

    try {
        $ready = Invoke-JsonRequest -Method "POST" -Uri "$BaseUrl/v1/delivery-dispatch/readiness" -Body @{
            event_id = $EventId
        }
        return [pscustomobject]@{
            blocked = $false
            status_code = 200
            message = $ready.data.reason
            error_kind = $null
            raw = $ready
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
        return [pscustomobject]@{
            blocked = ($statusCode -eq 400 -and $message -eq $ExpectedMessage -and $errorKind -eq "route_error")
            status_code = $statusCode
            message = $message
            error_kind = $errorKind
            raw = $null
        }
    }
}

$assets = New-SmokeAssets
$readiness = Invoke-JsonRequest -Method "POST" -Uri "$RuntimeBaseUrl/v1/outbound-cutover/readiness"
$queueBackend = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/queue-backend"

$cases = @(
    [pscustomobject]@{
        name = "first_account_private_file"
        mode = "state"
        expected = "succeeded"
        event_id = "qq-outbox-scope:first-private-file:{0}" -f ([guid]::NewGuid().ToString("N"))
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $FirstAccountId
            conversation_id = $FirstAccountPrivatePeerId
            conversation_type = "private"
        }
        content = "GO_OUTBOX_SCOPE_FIRST_PRIVATE_FILE_{0}" -f ([guid]::NewGuid().ToString("N").Substring(0, 8))
        attachments = @(
            @{
                kind = "file"
                url = $assets.file_path
                name = "scope-smoke.txt"
                mime_type = "text/plain"
            }
        )
        condition = { param($state) $state.status -eq "succeeded" }
    },
    [pscustomobject]@{
        name = "first_account_group_text_blocked"
        mode = "readiness"
        expected = "qq group sends are disabled"
        event_id = "qq-outbox-scope:first-group-text:{0}" -f ([guid]::NewGuid().ToString("N"))
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $FirstAccountId
            conversation_id = $FirstAccountGroupId
            conversation_type = "group"
        }
        content = "GO_OUTBOX_SCOPE_FIRST_GROUP_TEXT_{0}" -f ([guid]::NewGuid().ToString("N").Substring(0, 8))
        attachments = @()
    },
    [pscustomobject]@{
        name = "second_account_group_file_blocked"
        mode = "readiness"
        expected = "qq group sends are disabled"
        event_id = "qq-outbox-scope:second-group-file:{0}" -f ([guid]::NewGuid().ToString("N"))
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $SecondAccountId
            conversation_id = $SecondAccountGroupId
            conversation_type = "group"
        }
        content = "GO_OUTBOX_SCOPE_SECOND_GROUP_FILE_{0}" -f ([guid]::NewGuid().ToString("N").Substring(0, 8))
        attachments = @(
            @{
                kind = "file"
                url = $assets.file_path
                name = "scope-smoke.txt"
                mime_type = "text/plain"
            }
        )
    },
    [pscustomobject]@{
        name = "first_account_private_image_gated"
        mode = "state"
        expected = "queued"
        event_id = "qq-outbox-scope:first-private-image:{0}" -f ([guid]::NewGuid().ToString("N"))
        channel = @{
            kind = "qq"
            platform = "qq"
            account_id = $FirstAccountId
            conversation_id = $FirstAccountPrivatePeerId
            conversation_type = "private"
        }
        content = "GO_OUTBOX_SCOPE_FIRST_PRIVATE_IMAGE_{0}" -f ([guid]::NewGuid().ToString("N").Substring(0, 8))
        attachments = @(
            @{
                kind = "image"
                url = $assets.image_path
                name = "scope-smoke.png"
                mime_type = "image/png"
            }
        )
        condition = { param($state) $state.status -eq "queued" -and [int]$state.attempts -eq 0 -and $null -eq $state.leased_by }
    }
)

$results = foreach ($case in $cases) {
    $null = Submit-OutboxEvent -BaseUrl $RuntimeBaseUrl -EventId $case.event_id -Channel $case.channel -Content $case.content -Attachments $case.attachments -Metadata @{
        source = "verify_go_outbox_scope_live"
        live_smoke = "true"
        max_attempts = "1"
    }

    if ($case.mode -eq "readiness") {
        $readinessProbe = Test-DispatchReadinessBlocked -BaseUrl $RuntimeBaseUrl -EventId $case.event_id -ExpectedMessage $case.expected
        [pscustomobject]@{
            name = $case.name
            mode = $case.mode
            expected = $case.expected
            event_id = $case.event_id
            condition_met = $readinessProbe.blocked
            final_state = Get-OutboxState -BaseUrl $RuntimeBaseUrl -EventId $case.event_id
            history = @()
            readiness_probe = $readinessProbe
        }
        continue
    }

    $observed = Wait-ForOutboxState -BaseUrl $RuntimeBaseUrl -EventId $case.event_id -Condition $case.condition -TimeoutSeconds $PollSeconds
    [pscustomobject]@{
        name = $case.name
        mode = $case.mode
        expected = $case.expected
        event_id = $case.event_id
        condition_met = $observed.condition_met
        final_state = $observed.state
        history = $observed.history
        readiness_probe = $null
    }
}

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    readiness = $readiness.data
    queue_backend = [pscustomobject]@{
        outbox_execution_owner = $queueBackend.data.outbox_execution_owner
        outbox_execution_scope = $queueBackend.data.outbox_execution_scope
        outbox_allowed_kinds = $queueBackend.data.outbox_allowed_kinds
        outbox_allowed_kinds_by_account = $queueBackend.data.outbox_allowed_kinds_by_account
        outbox_allowed_kinds_by_account_conversation_type = $queueBackend.data.outbox_allowed_kinds_by_account_conversation_type
    }
    temp_assets = $assets
    cases = $results
} | ConvertTo-Json -Depth 12
