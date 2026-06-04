param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$TelegramApiBaseUrl = "",
    [string]$TelegramBotToken = "",
    [string]$RepoRoot = ""
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

function Resolve-TelegramToken {
    param([string]$ExplicitToken)

    if (-not [string]::IsNullOrWhiteSpace($ExplicitToken)) {
        return $ExplicitToken
    }
    if (-not [string]::IsNullOrWhiteSpace($env:TELEGRAM_BOT_TOKEN)) {
        return $env:TELEGRAM_BOT_TOKEN
    }
    if (-not [string]::IsNullOrWhiteSpace($env:AKASHIC_TELEGRAM_BOT_TOKEN)) {
        return $env:AKASHIC_TELEGRAM_BOT_TOKEN
    }
    return ""
}

function Get-TelegramConfigEvidence {
    param([string]$RootPath)

    if ([string]::IsNullOrWhiteSpace($RootPath)) {
        return [pscustomobject]@{
            repo_root = $null
            config_toml_exists = $false
            config_declares_telegram_channel = $false
            config_token_line = $null
            config_uses_env_placeholder = $false
            repo_env_exists = $false
            repo_env_has_telegram_key = $false
            codex_env_exists = $false
            codex_env_has_telegram_key = $false
        }
    }

    $configTomlPath = Join-Path $RootPath "config.toml"
    $repoEnvPath = Join-Path $RootPath ".env"
    $codexEnvPath = Join-Path $env:USERPROFILE ".codex\.env"

    $configText = ""
    $configExists = Test-Path -LiteralPath $configTomlPath
    if ($configExists) {
        $configText = Get-Content -LiteralPath $configTomlPath -Raw
    }

    $tokenLine = $null
    if ($configText -match '(?ms)^\[channels\.telegram\].*?^token\s*=\s*"(?<token>[^"]*)"') {
        $tokenLine = $matches.token
    }

    $repoEnvHasTelegramKey = $false
    if (Test-Path -LiteralPath $repoEnvPath) {
        $repoEnvHasTelegramKey = [bool](Select-String -Path $repoEnvPath -Pattern '^\s*(TELEGRAM_BOT_TOKEN|AKASHIC_TELEGRAM_BOT_TOKEN)\s*=' -SimpleMatch:$false -ErrorAction SilentlyContinue)
    }

    $codexEnvHasTelegramKey = $false
    if (Test-Path -LiteralPath $codexEnvPath) {
        $codexEnvHasTelegramKey = [bool](Select-String -Path $codexEnvPath -Pattern '^\s*(TELEGRAM_BOT_TOKEN|AKASHIC_TELEGRAM_BOT_TOKEN)\s*=' -SimpleMatch:$false -ErrorAction SilentlyContinue)
    }

    return [pscustomobject]@{
        repo_root = $RootPath
        config_toml_exists = $configExists
        config_declares_telegram_channel = ($configText -match '(?m)^\[channels\.telegram\]')
        config_token_line = $tokenLine
        config_uses_env_placeholder = ($tokenLine -eq '${TELEGRAM_BOT_TOKEN}' -or $tokenLine -eq '${AKASHIC_TELEGRAM_BOT_TOKEN}')
        repo_env_exists = (Test-Path -LiteralPath $repoEnvPath)
        repo_env_has_telegram_key = $repoEnvHasTelegramKey
        codex_env_exists = (Test-Path -LiteralPath $codexEnvPath)
        codex_env_has_telegram_key = $codexEnvHasTelegramKey
    }
}

$runtimeConfig = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/runtime-config"
$receiverStatuses = Invoke-JsonRequest -Method "GET" -Uri "$RuntimeBaseUrl/v1/receiver-statuses"

$resolvedToken = Resolve-TelegramToken -ExplicitToken $TelegramBotToken
$tokenPresent = -not [string]::IsNullOrWhiteSpace($resolvedToken)
$configEvidence = Get-TelegramConfigEvidence -RootPath $RepoRoot

$resolvedApiBase = if (-not [string]::IsNullOrWhiteSpace($TelegramApiBaseUrl)) {
    $TelegramApiBaseUrl
} elseif (-not [string]::IsNullOrWhiteSpace($env:AKASHIC_TELEGRAM_API_BASE_URL)) {
    $env:AKASHIC_TELEGRAM_API_BASE_URL
} elseif (-not [string]::IsNullOrWhiteSpace($runtimeConfig.data.delivery.telegram_endpoint)) {
    $runtimeConfig.data.delivery.telegram_endpoint
} else {
    "https://api.telegram.org"
}

$telegramReceivers = @($receiverStatuses.data.receivers | Where-Object { $_.kind -eq "telegram" })
$getMe = $null
$getMeError = $null
$backendState = "token_missing"

if ($tokenPresent) {
    try {
        $getMe = Invoke-RestMethod -Method "GET" -Uri ($resolvedApiBase.TrimEnd("/") + "/bot" + $resolvedToken + "/getMe") -TimeoutSec 30
        if ($getMe.ok -eq $true) {
            $backendState = "getme_ok"
        } else {
            $backendState = "getme_failed"
            $getMeError = "telegram_getme_not_ok"
        }
    }
    catch {
        $backendState = "getme_failed"
        $getMeError = $_.Exception.Message
        if (-not [string]::IsNullOrWhiteSpace($_.ErrorDetails.Message)) {
            $getMeError = $_.ErrorDetails.Message
        }
    }
}

[pscustomobject]@{
    runtime_base_url = $RuntimeBaseUrl
    telegram_api_base_url = $resolvedApiBase
    runtime_delivery = [pscustomobject]@{
        telegram_endpoint = $runtimeConfig.data.delivery.telegram_endpoint
        telegram_token_configured = $runtimeConfig.data.delivery.telegram_token_configured
    }
    runtime_receivers = [pscustomobject]@{
        telegram_receivers = $telegramReceivers
        totals = $receiverStatuses.data.totals
    }
    config_evidence = $configEvidence
    checks = [pscustomobject]@{
        runtime_reports_token_configured = [bool]$runtimeConfig.data.delivery.telegram_token_configured
        local_env_token_present = $tokenPresent
        telegram_receiver_present = (@($telegramReceivers).Count -gt 0)
        getme_attempted = $tokenPresent
        getme_ok = ($null -ne $getMe -and $getMe.ok -eq $true)
        backend_state = $backendState
    }
    get_me = if ($null -ne $getMe) {
        [pscustomobject]@{
            ok = $getMe.ok
            result = $getMe.result
        }
    } else {
        $null
    }
    get_me_error = $getMeError
} | ConvertTo-Json -Depth 12
