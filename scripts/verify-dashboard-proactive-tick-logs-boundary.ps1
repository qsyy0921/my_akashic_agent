param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pythonScript = Join-Path $scriptDir "verify_dashboard_proactive_tick_logs_boundary.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $scriptDir
}

Push-Location $RepoRoot
try {
    uv run python $pythonScript --repo-root $RepoRoot
}
finally {
    Pop-Location
}
