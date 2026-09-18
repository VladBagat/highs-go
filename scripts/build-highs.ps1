# Build the same pinned HiGHS release as the Dockerfile, using native MinGW.
[CmdletBinding()]
param(
    [string]$ToolchainBin,
    [ValidateRange(1, 128)][int]$Jobs = 4
)
$ErrorActionPreference = 'Stop'
$ToolchainBin = & "$PSScriptRoot/find-toolchain.ps1" -ToolchainBin $ToolchainBin
$repo = Split-Path $PSScriptRoot -Parent
$nativeRoot = Join-Path $repo '.native'
$source = Join-Path $nativeRoot 'src'
$build = Join-Path $nativeRoot 'build'
$install = Join-Path $nativeRoot 'highs'
$commit = '04024d701f79feb8e2f18bc3df0dffc04ef05088'

foreach ($tool in @('gcc.exe', 'g++.exe', 'mingw32-make.exe')) {
    if (!(Test-Path (Join-Path $ToolchainBin $tool))) {
        throw "Missing $tool in $ToolchainBin. Install MinGW tools; see README.md."
    }
}
$previousPath = $env:PATH
try {
    $env:PATH = "$ToolchainBin;$env:PATH"
    foreach ($tool in @('git', 'cmake')) { Get-Command $tool -ErrorAction Stop | Out-Null }
    $target = & gcc -dumpmachine
    if ($LASTEXITCODE -ne 0 -or $target -ne 'x86_64-w64-mingw32') {
        throw 'This script requires a native Windows x64 MinGW compiler.'
    }
    if (!(Test-Path $source)) {
        & git clone --depth 1 --branch v1.15.1 https://github.com/ERGO-Code/HiGHS.git $source
        if ($LASTEXITCODE -ne 0) { throw 'HiGHS download failed.' }
    }
    $actual = & git -C $source rev-parse HEAD
    if ($LASTEXITCODE -ne 0 -or $actual -ne $commit) { throw 'HiGHS source commit does not match the pinned release.' }
    $dirty = & git -C $source status --porcelain
    if ($LASTEXITCODE -ne 0 -or $dirty) { throw 'HiGHS source has local changes. Use a clean pinned checkout.' }

    & cmake -S $source -B $build -G 'MinGW Makefiles' `
        '-DCMAKE_C_COMPILER=gcc.exe' '-DCMAKE_CXX_COMPILER=g++.exe' `
        '-DCMAKE_BUILD_TYPE=Release' "-DCMAKE_INSTALL_PREFIX=$install" `
        '-DCMAKE_INSTALL_LIBDIR=lib' '-DFAST_BUILD=ON' '-DBUILD_SHARED_LIBS=ON' `
        '-DBUILD_SHARED_EXTRAS_LIB=OFF' '-DBUILD_CXX_EXE=OFF' `
        '-DBUILD_EXAMPLES=OFF' '-DBUILD_TESTING=OFF' '-DZLIB=OFF' '-DHIPO=OFF' `
        '-DHIGHSINT64=OFF'
    if ($LASTEXITCODE -ne 0) { throw 'HiGHS configure failed.' }
    & cmake --build $build --parallel $Jobs
    if ($LASTEXITCODE -ne 0) { throw 'HiGHS build failed.' }
    & cmake --install $build
    if ($LASTEXITCODE -ne 0) { throw 'HiGHS install failed.' }
    $notices = Join-Path $install 'share/doc/HIGHS'
    New-Item -ItemType Directory -Force $notices | Out-Null
    Copy-Item -LiteralPath (Join-Path $source 'LICENSE.txt') -Destination $notices
    Copy-Item -LiteralPath (Join-Path $source 'THIRD_PARTY_NOTICES.md') -Destination $notices
    Write-Host "HiGHS installed in $install"
} finally {
    $env:PATH = $previousPath
}
