$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if (-not $IsWindows) {
    throw 'This smoke test requires native Windows.'
}

$repo = Split-Path -Parent $PSScriptRoot
$gitBash = Join-Path $env:ProgramFiles 'Git\bin\bash.exe'
if (-not (Test-Path -LiteralPath $gitBash)) {
    throw "Git Bash was not found at $gitBash"
}

foreach ($tool in @('go', 'java', 'rg')) {
    Get-Command $tool -ErrorAction Stop | Out-Null
}

# Use Git's sh/cat for the documented PAGER workflow, never WSL's bash.
$previousPath = $env:PATH
$env:PATH = "$(Join-Path $env:ProgramFiles 'Git\usr\bin');$env:PATH"
$previousSmoke = $env:KSRC_WINDOWS_SMOKE
$previousBash = $env:KSRC_WINDOWS_GIT_BASH
Push-Location $repo
try {
    $env:KSRC_WINDOWS_SMOKE = '1'
    $env:KSRC_WINDOWS_GIT_BASH = $gitBash
    go test -count=1 -v -timeout 20m ./internal/windowssmoke
    if ($LASTEXITCODE -ne 0) {
        throw "Windows smoke tests failed (exit $LASTEXITCODE); see command stdout/stderr above."
    }
} finally {
    $env:PATH = $previousPath
    $env:KSRC_WINDOWS_SMOKE = $previousSmoke
    $env:KSRC_WINDOWS_GIT_BASH = $previousBash
    Pop-Location
}
