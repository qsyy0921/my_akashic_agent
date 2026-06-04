param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$PrimaryAccountId = "1049511700",
    [string]$SecondaryAccountId = "2365524513"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pythonScript = Join-Path $scriptDir "verify_qq_cutover_route_matrix.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

$repoRoot = Split-Path -Parent $scriptDir
Push-Location $repoRoot
try {
    uv run python $pythonScript `
        --runtime-base-url $RuntimeBaseUrl `
        --primary-account-id $PrimaryAccountId `
        --secondary-account-id $SecondaryAccountId
}
finally {
    Pop-Location
}
