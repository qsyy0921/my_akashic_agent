param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $PSScriptRoot
}

$pythonExe = "python"
if (Get-Command uv -ErrorAction SilentlyContinue) {
    uv run python (Join-Path $PSScriptRoot "verify_agent_worker_status_heartbeat_live_smoke.py") --repo-root $RepoRoot
    exit $LASTEXITCODE
}

& $pythonExe (Join-Path $PSScriptRoot "verify_agent_worker_status_heartbeat_live_smoke.py") --repo-root $RepoRoot
exit $LASTEXITCODE
