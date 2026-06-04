param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [string]$DashboardBaseUrl = "http://127.0.0.1:2236"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pythonScript = Join-Path $scriptDir "verify_dashboard_knowledge_rag_state_boundary.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

$repoRoot = Split-Path -Parent $scriptDir
Push-Location $repoRoot
try {
    uv run python $pythonScript --runtime-base-url $RuntimeBaseUrl --dashboard-base-url $DashboardBaseUrl
}
finally {
    Pop-Location
}
