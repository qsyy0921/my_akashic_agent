param(
    [string]$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path,
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780"
)

$ErrorActionPreference = "Stop"

$pythonExe = if ($env:VIRTUAL_ENV) {
    Join-Path $env:VIRTUAL_ENV "Scripts\python.exe"
} else {
    "python"
}

$pythonScript = Join-Path $PSScriptRoot "verify_agent_job_external_lease_cutover_diff.py"

& $pythonExe $pythonScript --repo-root $RepoRoot --runtime-base-url $RuntimeBaseUrl
