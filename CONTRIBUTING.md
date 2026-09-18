# Contributing

Development uses Go 1.26 or later and HiGHS v1.15.1. The build scripts pin the upstream commit and use 32-bit `HighsInt`, with HiPO disabled.

## Windows development


Run the build commands in PowerShell. Install compiler packages in the MSYS2 UCRT64 terminal.

### 1. Install build tools

Install [Go](https://go.dev/dl/), [Git](https://git-scm.com/downloads/win), [CMake](https://cmake.org/download/), and [MSYS2](https://www.msys2.org/). Make Go, Git, and CMake available on `PATH` using their installers, then open a new terminal.

In the **MSYS2 UCRT64** terminal, update MSYS2 (follow any restart instructions):

```sh
pacman -Syu
pacman -S --needed mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-make
```

Use a Windows x64 MinGW toolchain consistently for HiGHS and Go. An existing MinGW installation can be supplied through `-ToolchainBin`.

### 2. Build HiGHS

From the root of this checkout in PowerShell:

```powershell
./scripts/build-highs.ps1
```

The script downloads the pinned HiGHS source and builds it under `.native/`, which is ignored by Git. It installs the DLL, import library, headers, and upstream notices under `.native/highs`. Build outputs and dependencies stay under `.native/`.

The default compiler directory is `C:\msys64\ucrt64\bin`. To select another installation:

```powershell
./scripts/build-highs.ps1 -ToolchainBin 'C:\path\to\mingw\bin'
```

Use the same toolchain for HiGHS and Go. If you change compilers, use a fresh `.native/build` directory to avoid reusing CMake's old compiler settings.

### 3. Configure the terminal and run

For VS Code, run `./scripts/setup.ps1` once and open the generated `highs-go.code-workspace`. It configures Go diagnostics and new terminals and reuses the HiGHS build above. The same helper works from a consuming application's directory. See [installation and application distribution](README.md#installation).

For a standalone PowerShell terminal:

```powershell
. ./scripts/enter-dev.ps1
go test -count=1 ./...
go vet ./...
go run ./cmd/example
```

The leading dot and space on the first line are intentional. If you selected another compiler, pass the same `-ToolchainBin` to `enter-dev.ps1`.

Expected example output:

```text
status=optimal
objective=2 x=4 y=2
```

Repeat the `enter-dev.ps1` command in each new terminal. It sets `CGO_ENABLED`, `CC`, `CXX`, `CGO_CFLAGS`, and `CGO_LDFLAGS`, and adds the HiGHS and compiler DLL directories to this terminal's `PATH`. It replaces existing cgo flag values in that session.

To build a standalone executable:

```powershell
go build -o highs-example.exe ./cmd/example
./highs-example.exe
```

To distribute it without requiring environment setup on the target machine:

```powershell
./scripts/bundle-windows.ps1 -Executable ./highs-example.exe -Destination ./dist/highs-example
```

Distribute the entire folder. `./scripts/test-consumption.ps1` checks a separate consumer module and runs its bundle with only Windows directories on `PATH`; Windows CI runs this check too.

If PowerShell blocks local scripts, review them and use `Set-ExecutionPolicy -Scope Process Bypass` for that terminal only, where your machine's policy allows it.

### Common setup problems

| Error | Check |
| --- | --- |
| `gcc` not found | Install the toolchain and pass its `bin` directory to both scripts. |
| `highs_c_api.h` not found | Build HiGHS, then run `enter-dev.ps1` in the current terminal. |
| `cannot find -lhighs` | Check `.native/highs/lib/libhighs.dll.a` and `CGO_LDFLAGS`. |
| DLL missing or exit code `0xc0000135` | Run from the configured terminal; both HiGHS and MinGW runtime DLL directories must be on `PATH`. |
| CMake generator/compiler mismatch | Start with a fresh `.native/build` directory after switching toolchains. |

## Linux / Docker

From the repository root:

```sh
docker build -t highs-go .
docker run --rm highs-go
```

The Docker build compiles HiGHS, runs the Go tests, and builds the example. The runtime image includes the native library and notices. Expected output is `status=optimal` followed by `objective=2 x=4 y=2`.

## Testing changes

With the native environment configured:

```sh
go test -count=1 ./...
go vet ./...
go run ./cmd/example
```

Tests exercise real HiGHS solves, including expected primal and dual results, maximization, equality constraints, infeasible and unbounded models, empty rows, input validation, and sparse matrix packing. Add tests for changes to the exposed behavior and compare numerical results with tolerances.

Format Go changes with `gofmt`. Keep the public API and its documentation consistent. GitHub Actions builds Windows x64 with MSYS2 UCRT64 and Linux through Docker.

## Testing a local dependency

In a consuming project's directory, point the module at a local checkout:

```powershell
go mod edit '-require=github.com/VladBagat/highs-go@v0.0.0'
go mod edit '-replace=github.com/VladBagat/highs-go=C:/src/highs-go'
```

Adjust the checkout path and configure the native environment in the same terminal before building. On Windows, dot-source the checkout's `scripts/enter-dev.ps1`, passing `-ToolchainBin` if needed. Remove the local replacement before testing a published version.
