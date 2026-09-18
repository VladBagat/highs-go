# Run once for each consuming project. Open the generated workspace in VS Code.
[CmdletBinding()]
param(
    [string]$ProjectPath = (Get-Location).Path,
    [string]$ToolchainBin,
    [string]$HighsRoot = (Join-Path (Split-Path $PSScriptRoot -Parent) '.native/highs')
)
$ErrorActionPreference = 'Stop'
$ProjectPath = (Resolve-Path -LiteralPath $ProjectPath).Path
$ToolchainBin = & "$PSScriptRoot/find-toolchain.ps1" -ToolchainBin $ToolchainBin
if (!(Test-Path (Join-Path $HighsRoot 'lib/libhighs.dll.a'))) {
    $defaultRoot = Join-Path (Split-Path $PSScriptRoot -Parent) '.native/highs'
    if ([IO.Path]::GetFullPath($HighsRoot) -ne [IO.Path]::GetFullPath($defaultRoot)) {
        throw 'The supplied -HighsRoot has no MinGW HiGHS installation.'
    }
    & "$PSScriptRoot/build-highs.ps1" -ToolchainBin $ToolchainBin
}
# Reuse the tested terminal configuration without changing the calling shell.
$names = @('PATH', 'CGO_ENABLED', 'CC', 'CXX', 'CGO_CFLAGS', 'CGO_LDFLAGS')
$saved = @{}
$environment = [ordered]@{}
try {
    foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
    . "$PSScriptRoot/enter-dev.ps1" -ToolchainBin $ToolchainBin -HighsRoot $HighsRoot
    foreach ($name in $names) { $environment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process') }
    Push-Location (Split-Path $PSScriptRoot -Parent)
    try {
        go run ./cmd/example
        if ($LASTEXITCODE -ne 0) { throw 'HiGHS build/run check failed. Check the compiler and native installation above.' }
    } finally { Pop-Location }
} finally {
    foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
}
$workspace = Join-Path $ProjectPath 'highs-go.code-workspace'
if (Test-Path $workspace) {
    $config = Get-Content -LiteralPath $workspace -Raw | ConvertFrom-Json
} else {
    $config = [pscustomobject]@{ folders = @(@{ path = '.' }); settings = [pscustomobject]@{} }
}
if (!$config.settings) { $config | Add-Member -NotePropertyName settings -NotePropertyValue ([pscustomobject]@{}) -Force }
foreach ($setting in @('go.toolsEnvVars', 'terminal.integrated.env.windows')) {
    if (!$config.settings.$setting) {
        $config.settings | Add-Member -NotePropertyName $setting -NotePropertyValue ([pscustomobject]@{}) -Force
    }
    foreach ($name in $names) {
        $config.settings.$setting | Add-Member -NotePropertyName $name -NotePropertyValue $environment[$name] -Force
    }
}
$config | ConvertTo-Json -Depth 20 | Set-Content -LiteralPath $workspace -Encoding utf8
Write-Host "Open $workspace in VS Code. New terminals and Go tools are configured automatically."
