param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $PSScriptRoot
}

$scriptPath = Join-Path $PSScriptRoot "verify_agent_worker_status_startup_prune_live_smoke.py"

if (Get-Command uv -ErrorAction SilentlyContinue) {
    uv run python $scriptPath --repo-root $RepoRoot
    exit $LASTEXITCODE
}

& python $scriptPath --repo-root $RepoRoot
exit $LASTEXITCODE
