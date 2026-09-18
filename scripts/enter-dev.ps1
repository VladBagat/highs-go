# Dot-source this script to configure only the current PowerShell session.
[CmdletBinding()]
param(
    [string]$ToolchainBin,
    [string]$HighsRoot = (Join-Path (Split-Path $PSScriptRoot -Parent) '.native/highs')
)
$ToolchainBin = & "$PSScriptRoot/find-toolchain.ps1" -ToolchainBin $ToolchainBin
foreach ($tool in @('gcc.exe', 'g++.exe')) {
    if (!(Test-Path (Join-Path $ToolchainBin $tool))) { throw "Missing $tool in $ToolchainBin." }
}
if (!(Test-Path (Join-Path $HighsRoot 'include/highs/interfaces/highs_c_api.h'))) {
    throw 'HiGHS headers not found. Run scripts/build-highs.ps1 first.'
}
if (!(Test-Path (Join-Path $HighsRoot 'lib/libhighs.dll.a'))) {
    throw 'MinGW HiGHS import library not found. Run scripts/build-highs.ps1 first.'
}
if (!(Get-ChildItem (Join-Path $HighsRoot 'bin') -Filter '*highs*.dll' -ErrorAction SilentlyContinue)) {
    throw 'HiGHS DLL not found. Run scripts/build-highs.ps1 first.'
}
$HighsRoot = (Resolve-Path $HighsRoot).Path
$ToolchainBin = (Resolve-Path $ToolchainBin).Path
$highsPath = $HighsRoot.Replace('\', '/')
$env:PATH = "$(Join-Path $HighsRoot 'bin');$ToolchainBin;$env:PATH"
$env:CGO_ENABLED = '1'
$env:CC = 'gcc.exe'
$env:CXX = 'g++.exe'
# Quotes keep paths containing spaces together when cgo parses these flags.
$env:CGO_CFLAGS = '"-I' + $highsPath + '/include/highs"'
$env:CGO_LDFLAGS = '"-L' + $highsPath + '/lib"'
Write-Host 'HiGHS is configured for this terminal. Run: go test -count=1 ./...'
