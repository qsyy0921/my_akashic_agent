param(
    [string]$RepoRoot = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($RepoRoot)) {
    $RepoRoot = Split-Path -Parent $PSScriptRoot
}

$pythonScript = Join-Path $PSScriptRoot "verify_media_asset_content_recovery_live_smoke.py"
if (-not (Test-Path $pythonScript)) {
    throw "python verifier not found: $pythonScript"
}

if (Get-Command uv -ErrorAction SilentlyContinue) {
    uv run python $pythonScript --repo-root $RepoRoot
    exit $LASTEXITCODE
}

python $pythonScript --repo-root $RepoRoot
exit $LASTEXITCODE
