param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $scriptDir
}

$pythonScript = Join-Path $scriptDir "verify_agent_job_external_lease_nats_smoke.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

Push-Location $RepoRoot
try {
    uv run python $pythonScript --repo-root $RepoRoot
}
finally {
    Pop-Location
}
