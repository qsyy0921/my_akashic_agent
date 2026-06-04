param(
    [string]$RuntimeBaseUrl = "http://127.0.0.1:8780"
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$pythonScript = Join-Path $scriptDir "verify_queue_topology_boundary.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

$repoRoot = Split-Path -Parent $scriptDir
Push-Location $repoRoot
try {
    uv run python $pythonScript --runtime-base-url $RuntimeBaseUrl
}
finally {
    Pop-Location
}
