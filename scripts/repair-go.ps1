<#
 Attempts deeper repair steps for broken Go stdlib/toolchain resolution.
 Use when doctor + build still show 'package X is not in std'.
#>
param()
Set-StrictMode -Version Latest
$ErrorActionPreference='Stop'

Write-Host '--- Go Repair Assistant ---' -ForegroundColor Cyan
go version
Write-Host "GOROOT: $(go env GOROOT)"

Write-Host '1) Unset explicit GOROOT in current session (if any)'
Remove-Item Env:GOROOT -ErrorAction SilentlyContinue

Write-Host '2) Clean all caches'
go clean -cache -modcache -testcache -fuzzcache

Write-Host '3) Remove downloaded toolchain modules (best effort)'
$gomodcache = go env GOMODCACHE
Get-ChildItem -Path (Join-Path $gomodcache 'golang.org') -Filter 'toolchain@*' -ErrorAction SilentlyContinue | ForEach-Object {
  Write-Host "Deleting $($_.FullName)"
  Remove-Item -Recurse -Force $_.FullName -ErrorAction SilentlyContinue
}

Write-Host '4) Force local toolchain usage'
go env -w GOTOOLCHAIN=local

Write-Host '5) Probe std list'
if (go list builtin 1>$null 2>$null) { Write-Host 'Std list OK' } else { Write-Warning 'Std list still failing (will continue)'}

Write-Host '6) If still broken: reinstall Go via winget (requires elevation):'
Write-Host '   winget upgrade --id GoLang.Go' -ForegroundColor Yellow

Write-Host '7) After reinstall open NEW shell and run: scripts/build.ps1' -ForegroundColor Yellow

Write-Host 'Repair assistant finished.' -ForegroundColor Green