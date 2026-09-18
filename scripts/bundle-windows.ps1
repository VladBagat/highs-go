# Bundle an already-built x64 application. Destination must be new to avoid stale DLLs.
[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$Executable,
    [Parameter(Mandatory)][string]$Destination,
    [string]$ToolchainBin,
    [string]$HighsRoot = (Join-Path (Split-Path $PSScriptRoot -Parent) '.native/highs')
)
$ErrorActionPreference = 'Stop'
$ToolchainBin = & "$PSScriptRoot/find-toolchain.ps1" -ToolchainBin $ToolchainBin
$Executable = (Resolve-Path -LiteralPath $Executable).Path
$HighsRoot = (Resolve-Path -LiteralPath $HighsRoot).Path
$objdump = Join-Path $ToolchainBin 'objdump.exe'
if (!(Test-Path $objdump)) { throw "Missing $objdump. Install MinGW binutils." }
if (Test-Path -LiteralPath $Destination) { throw 'Destination already exists. Choose a new directory.' }
$Destination = (New-Item -ItemType Directory -Path $Destination).FullName
Copy-Item -LiteralPath $Executable -Destination $Destination
$pending = [Collections.Generic.Queue[string]]::new()
$pending.Enqueue($Executable)
$seen = @{}
while ($pending.Count) {
    $binary = $pending.Dequeue()
    $imports = & $objdump -p $binary
    if ($LASTEXITCODE -ne 0) { throw "Cannot inspect $binary." }
    foreach ($line in $imports) {
        if ($line -notmatch 'DLL Name:\s*(\S+)') { continue }
        $dll = $Matches[1]
        if ($seen.ContainsKey($dll)) { continue }
        $seen[$dll] = $true
        # API sets are supplied by Windows. Prefer app/toolchain DLLs over
        # System32: a developer machine may have extra runtimes installed there.
        if ($dll -match '^(api|ext)-ms-') { continue }
        $source = $null
        foreach ($directory in @((Split-Path $Executable -Parent), (Join-Path $HighsRoot 'bin'), $ToolchainBin)) {
            $candidate = Join-Path $directory $dll
            if (Test-Path -LiteralPath $candidate) { $source = $candidate; break }
        }
        if (!$source -and (Test-Path (Join-Path "$env:SystemRoot/System32" $dll))) { continue }
        if (!$source) { throw "Cannot locate $dll required by $binary." }
        Copy-Item -LiteralPath $source -Destination $Destination
        $pending.Enqueue($source)
    }
}
$licenses = New-Item -ItemType Directory -Path (Join-Path $Destination 'licenses')
Copy-Item -LiteralPath (Join-Path $HighsRoot 'share/doc/HIGHS') -Destination (Join-Path $licenses.FullName 'HiGHS') -Recurse
Copy-Item -LiteralPath (Join-Path (Split-Path $PSScriptRoot -Parent) 'LICENSE') -Destination (Join-Path $licenses.FullName 'highs-go.txt')
# Preserve toolchain notices as supplied; extra notices are harmless.
Copy-Item -LiteralPath (Join-Path (Split-Path $ToolchainBin -Parent) 'share/licenses') -Destination (Join-Path $licenses.FullName 'toolchain') -Recurse
Write-Host "Bundle ready: $Destination. Distribute the whole directory (plus your application's own assets/licenses)."
