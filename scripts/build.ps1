param(
    [switch]$Release
)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$bin = Join-Path $PSScriptRoot '..' 'bin'
$lib = Join-Path $PSScriptRoot '..' 'lib'
if(!(Test-Path $bin)){ New-Item -ItemType Directory -Path $bin | Out-Null }

Write-Host 'Building fit (libfido2 hardware CLI)...'
go build -o "$bin/fit.exe" ./cmd/fit
Write-Host 'Building fit-hello (Windows Hello CLI)...'
go build -o "$bin/fit-hello.exe" ./cmd/fit-hello

Write-Host 'Copying runtime libraries...'
Get-ChildItem -Path $lib -Filter *.dll | Copy-Item -Destination $bin -Force

Write-Host 'Contents of bin:'
Get-ChildItem $bin | Select-Object Name,Length | Format-Table -AutoSize

Write-Host 'Done.'
