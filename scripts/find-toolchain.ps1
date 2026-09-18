# Shared by the Windows setup/build scripts. Prefer an explicitly selected compiler.
param([string]$ToolchainBin)
if (!$ToolchainBin) {
    $compiler = Get-Command gcc.exe -ErrorAction SilentlyContinue
    if ($compiler) { $ToolchainBin = Split-Path $compiler.Source -Parent }
    elseif (Test-Path 'C:/msys64/ucrt64/bin/gcc.exe') { $ToolchainBin = 'C:/msys64/ucrt64/bin' }
    else { throw 'Install MSYS2 UCRT64 GCC, or supply -ToolchainBin pointing to your MinGW bin directory.' }
}
$ToolchainBin = (Resolve-Path -LiteralPath $ToolchainBin -ErrorAction Stop).Path
foreach ($tool in @('gcc.exe', 'g++.exe')) {
    if (!(Test-Path (Join-Path $ToolchainBin $tool))) { throw "Missing $tool in $ToolchainBin." }
}
$target = & (Join-Path $ToolchainBin 'gcc.exe') -dumpmachine
if ($LASTEXITCODE -ne 0 -or $target -ne 'x86_64-w64-mingw32') { throw 'A Windows x64 MinGW compiler is required.' }
$ToolchainBin
