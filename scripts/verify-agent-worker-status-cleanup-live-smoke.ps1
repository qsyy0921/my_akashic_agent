param(
    [string]$RepoRoot = "",
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780",
    [switch]$ApplyLiveRuntimeCleanup
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $PSScriptRoot
}

$scriptPath = Join-Path $PSScriptRoot "verify_agent_worker_status_cleanup_live_smoke.py"
$arguments = @("--repo-root", $RepoRoot, "--runtime-base-url", $RuntimeBaseUrl)
if ($ApplyLiveRuntimeCleanup) {
    $arguments += "--apply-live-runtime-cleanup"
}

if (Get-Command uv -ErrorAction SilentlyContinue) {
    uv run python $scriptPath @arguments
    exit $LASTEXITCODE
}

& python $scriptPath @arguments
exit $LASTEXITCODE
