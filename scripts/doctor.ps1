<#
 .SYNOPSIS
  Environment doctor for Go toolchain issues (std library not found / govulncheck failures).

 .DESCRIPTION
  Detects common misconfigurations causing errors like:
    package unsafe is not in std (...golang.org\toolchain@...\src\unsafe)

  Provides actionable remediation steps without modifying your system.
#>
param()
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Section($name){ Write-Host "`n=== $name ===" -ForegroundColor Cyan }

Section 'Go Version & Toolchain'
go version
Write-Host "GOTOOLCHAIN: $(go env GOTOOLCHAIN)"
$goroot = go env GOROOT
Write-Host "GOROOT: $goroot"
if([string]::IsNullOrWhiteSpace($goroot)) { Write-Warning 'GOROOT empty (let Go manage it automatically).'}

Section 'GOROOT Sanity'
if($goroot -match 'pkg[\\/ ]mod[\\/ ]golang.org[\\/ ]toolchain@') {
  Write-Warning 'GOROOT points inside module cache toolchain shim (golang.org/toolchain).'
  Write-Host 'This layout can appear if experimental downloaded toolchain is partially corrupted.' -ForegroundColor Yellow
}
if(Test-Path (Join-Path $goroot 'src' 'fmt')) { Write-Host 'fmt package path exists.' } else { Write-Warning 'fmt directory missing in GOROOT/src (corruption suspected).'}
if(Test-Path (Join-Path $goroot 'src' 'unsafe')) { Write-Host 'unsafe package path exists.' } else { Write-Warning 'unsafe directory missing in GOROOT/src (corruption suspected).'}

Section 'Std Package Probe'
$stdOk = $true
try { go list builtin 1>$null 2>$null } catch { $stdOk = $false }
if($stdOk){ Write-Host 'Std library listing succeeded.' } else { Write-Warning 'go list builtin failed (std resolution broken).'}

Section 'Module & Build Cache Summary'
Write-Host "GOMODCACHE: $(go env GOMODCACHE)"
Write-Host "GOCACHE   : $(go env GOCACHE)"

Section 'Test Minimal Build'
try {
  go build -o $env:TEMP/fit-env-check.exe ./cmd/fit 1>$null 2>$null
  if(Test-Path "$env:TEMP/fit-env-check.exe") { Write-Host 'Build succeeded.' }
} catch {
  Write-Warning 'Direct build failed (see remediation below).'
}

Section 'Remediation'
@'
If you saw warnings above:
 1. Remove any user/system GOROOT environment variable (let Go decide).
 2. Reinstall / repair Go (official installer or: winget upgrade --id GoLang.Go).
 3. Clear caches:
      go clean -cache -modcache -testcache -fuzzcache
      Remove-Item "$($env:GOMODCACHE)\golang.org\toolchain*" -Recurse -Force -ErrorAction SilentlyContinue
 4. (Optional) Force local toolchain: go env -w GOTOOLCHAIN=local
 5. Re-open shell and run: scripts/build.ps1

govulncheck relies on normal 'go list'; once std resolution works it will pass.
'@ | Write-Host

Write-Host "\nDoctor completed." -ForegroundColor Green
