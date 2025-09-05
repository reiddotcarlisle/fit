param(
    [switch]$Release
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$bin = Join-Path $PSScriptRoot '..' 'bin'
$lib = Join-Path $PSScriptRoot '..' 'lib'
if(!(Test-Path $bin)){ New-Item -ItemType Directory -Path $bin | Out-Null }

$version = (git describe --tags --always --dirty 2>$null); if(-not $version){ $version = 'dev' }
$ldflags = "-X 'main.buildVersion=$version'"

# Provide a safe default GOPROXY only if user hasn't set one.
if (-not $env:GOPROXY -or $env:GOPROXY -eq '') {
    $env:GOPROXY = 'https://proxy.golang.org,direct'
    Write-Host "GOPROXY not set; using default: $env:GOPROXY"
} else {
    Write-Host "GOPROXY already set: $env:GOPROXY"
}

# Detect broken stdlib resolution (common when GOROOT points into golang.org/toolchain shim but is corrupted)
$goroot = go env GOROOT
if ($goroot -match 'pkg[\\/ ]mod[\\/ ]golang.org[\\/ ]toolchain@') {
    try { go list builtin 1>$null 2>$null } catch { }
    if ($LASTEXITCODE -ne 0) {
        Write-Warning 'Std library resolution appears broken (golang.org/toolchain shim).'
        Write-Warning 'Run scripts/doctor.ps1 for diagnostics + remediation steps before continuing.'
        Write-Warning 'Skipping build to avoid noisy failures.'
        return
    }
}

Write-Host 'Running formatting check (gofmt)...'
$notFormatted = gofmt -l . | Where-Object { $_ -and ($_ -notmatch '^vendor/') }
if($notFormatted){ Write-Warning "Files need formatting:`n$($notFormatted -join "`n")" }

Write-Host 'Running go mod tidy (non-enforcing)...'
go mod tidy

Write-Host 'Running go vet...'
go vet ./... || Write-Warning 'go vet reported issues (non-fatal for local builds).'

Write-Host 'Checking for staticcheck...'
if (Get-Command staticcheck -ErrorAction SilentlyContinue) {
    Write-Host 'Running staticcheck...'
    $scOut = & staticcheck ./... 2>&1
    $scCode = $LASTEXITCODE
    if ($scCode -ne 0) {
        if ($scOut -match 'module requires at least go[0-9.]+, but Staticcheck was built with go[0-9.]+') {
            Write-Warning 'staticcheck binary built with older Go toolchain; re-installing...'
            go install honnef.co/go/tools/cmd/staticcheck@latest
            Write-Host 'Re-running staticcheck after reinstall...'
            & staticcheck ./... 2>&1 | Out-String | Write-Host
        } else {
            $scOut | Out-String | Write-Warning
            Write-Warning 'staticcheck reported issues.'
        }
    } else {
        $scOut | Out-String | Write-Host
    }
} else {
    Write-Host 'staticcheck not found (skip). Install with: go install honnef.co/go/tools/cmd/staticcheck@latest'
}

Write-Host 'Building fit (libfido2 hardware CLI)...'
go build -ldflags $ldflags -o "$bin/fit.exe" ./cmd/fit
Write-Host 'Building fit-hello (Windows Hello CLI)...'
go build -ldflags $ldflags -o "$bin/fit-hello.exe" ./cmd/fit-hello

Write-Host 'Copying runtime libraries...'
Get-ChildItem -Path $lib -Filter *.dll | Copy-Item -Destination $bin -Force

Write-Host 'Contents of bin:'
Get-ChildItem $bin | Select-Object Name,Length | Format-Table -AutoSize

Write-Host 'Optionally run vulnerability scan (govulncheck)'
if (Get-Command govulncheck -ErrorAction SilentlyContinue) {
    govulncheck ./... | Out-Host
} else {
    Write-Host 'govulncheck not installed (skip)'
}

Write-Host 'Done.'
