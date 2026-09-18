# Integration check: consume from another module, then run with only Windows on PATH.
[CmdletBinding()]
param([string]$ToolchainBin = 'C:\msys64\ucrt64\bin')
$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$scratch = Join-Path $repo ('.native/consumer test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $scratch | Out-Null
$modulePath = $repo.Replace('\', '/')
@"
module consumer

go 1.26

require github.com/VladBagat/highs-go v0.0.0
replace github.com/VladBagat/highs-go => "$modulePath"
"@ | Set-Content (Join-Path $scratch 'go.mod')
Copy-Item (Join-Path $repo 'cmd/example/main.go') (Join-Path $scratch 'main.go')
& "$PSScriptRoot/setup.ps1" -ProjectPath $scratch -ToolchainBin $ToolchainBin
$workspace = Get-Content (Join-Path $scratch 'highs-go.code-workspace') -Raw | ConvertFrom-Json
$workspace.settings | Add-Member -NotePropertyName 'editor.fontSize' -NotePropertyValue 17
$workspace | ConvertTo-Json -Depth 20 | Set-Content (Join-Path $scratch 'highs-go.code-workspace')
& "$PSScriptRoot/setup.ps1" -ProjectPath $scratch -ToolchainBin $ToolchainBin
$workspace = Get-Content (Join-Path $scratch 'highs-go.code-workspace') -Raw | ConvertFrom-Json
if ($workspace.settings.'editor.fontSize' -ne 17) { throw 'Setup lost existing workspace settings.' }
$saved = @{}
try {
    foreach ($property in $workspace.settings.'go.toolsEnvVars'.PSObject.Properties) {
        $saved[$property.Name] = [Environment]::GetEnvironmentVariable($property.Name, 'Process')
        [Environment]::SetEnvironmentVariable($property.Name, $property.Value, 'Process')
    }
    Push-Location $scratch
    try {
        go build -o app.exe .
        if ($LASTEXITCODE -ne 0) { throw 'Consumer build failed.' }
        & "$PSScriptRoot/bundle-windows.ps1" -Executable ./app.exe -Destination ./bundle -ToolchainBin $ToolchainBin
        $env:PATH = "$env:SystemRoot\System32;$env:SystemRoot"
        $output = & ./bundle/app.exe
        if ($LASTEXITCODE -ne 0 -or ($output -join "`n") -notmatch 'objective=2 x=4 y=2') {
            throw "Bundled application failed: $output"
        }
        if (!(Test-Path ./bundle/licenses/HiGHS/LICENSE.txt)) { throw 'Missing HiGHS license.' }
        if (!(Test-Path ./bundle/licenses/toolchain/gcc-libs)) { throw 'Missing runtime licenses.' }
    } finally { Pop-Location }
} finally {
    foreach ($name in $saved.Keys) { [Environment]::SetEnvironmentVariable($name, $saved[$name], 'Process') }
}
Write-Host 'Consumer build and standalone bundle passed.'
