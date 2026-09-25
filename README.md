# highs-go

A small Go wrapper around [HiGHS](https://highs.dev/) for continuous linear programming.

Built primarily for personal use, with AI-assisted development and AI-generated code. Use with caution and validate results for your application. This is an independent project with an experimental API that may change before v1.0.

## Features

- Continuous variables with lower and upper bounds.
- Minimization and maximization, including an objective offset.
- Equality, one-sided, and ranged linear constraints.
- Solver status, objective value, primal values, reduced costs, row activities, and row duals.

The API covers batch LP solving through the HiGHS C interface, with optional time limits and logging. Each solve creates and releases a native solver, with console output disabled. Integer and quadratic programming, algorithm tuning, cancellation, and warm starts are outside the current scope.

## Installation

```sh
go get github.com/VladBagat/highs-go
```

Requires **Go 1.26+**, **cgo**, a compatible C compiler, and **HiGHS v1.15.1** headers and shared library with 32-bit `HighsInt`. Install the native dependency separately and configure the build and runtime paths for its location.

### Automatic mode

Keep a checkout of this repository for the native dependency and setup helpers. Install the prerequisites below, then run setup from **your application's directory**. Setup builds the pinned HiGHS release locally when needed, verifies a real solve, and creates `highs-go.code-workspace` with settings for VS Code's Go tools and new integrated terminals.

Open that workspace on subsequent visits; no repeated environment commands are needed. Add `highs-go.code-workspace` to your application's `.gitignore` because it contains local paths. Rerunning setup preserves unrelated workspace settings. Unix setup accepts existing workspaces in JSON format, without comments or trailing commas.

#### Windows x64

Install [Go](https://go.dev/dl/), Git, CMake and [MSYS2](https://www.msys2.org/). In the MSYS2 **UCRT64** terminal, install the compiler once:

```sh
pacman -S --needed mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-make
```

From your application's directory in PowerShell:

```powershell
& C:/src/highs-go/scripts/setup.ps1
code ./highs-go.code-workspace
```

Replace `C:/src/highs-go` with your checkout. Compiler discovery uses `PATH`, then `C:/msys64/ucrt64/bin`. Use `-ToolchainBin C:/path/to/mingw/bin` to override it, or `-HighsRoot C:/highs` for an existing compatible installation. Use `-ProjectPath C:/path/to/app` to choose the workspace directory explicitly.

#### Linux

Install Go 1.26+, a C/C++ compiler, Make, Git, CMake, `pkg-config`, and Python 3. For example, on Debian/Ubuntu (install Go separately):

```sh
sudo apt-get install build-essential git cmake pkg-config python3
```

From your application's directory:

```sh
bash /path/to/highs-go/scripts/setup.sh
code ./highs-go.code-workspace
```

#### macOS

Install Go 1.26+ and the Xcode Command Line Tools (`xcode-select --install`), plus Git, CMake, `pkg-config`, and Python 3. With Homebrew, the additional tools can be installed with:

```sh
brew install cmake pkg-config python
```

From your application's directory (Bash or Zsh):

```sh
bash /path/to/highs-go/scripts/setup.sh
code ./highs-go.code-workspace
```

If `code` is unavailable, use VS Code's **File → Open Workspace from File**.

Linux and macOS share the same scripts. They install HiGHS under the checkout's `.native/highs`, without `sudo` or shell-profile changes. Use `--highs-root /path/to/highs` to reuse an existing compatible installation, `--project-path /path/to/app` to choose the workspace directory, and `--jobs 8` to change build parallelism (default: 4). Set `CC` and `CXX` to compiler executable paths before setup to override the default `cc` and `c++`; compiler command strings with extra arguments are not supported.

The generated environment uses `pkg-config` for build flags and embeds the HiGHS library directory in compiled binaries as a runtime search path. Keep that directory available when running them. Rerun setup and rebuild if you move the checkout or native installation. Use a fresh `.native/build` directory when switching compilers or architectures. Build natively for the architecture of your Go toolchain; these helpers do not cross-compile.

For another editor or an external terminal, configure each session using the matching helper:

```powershell
# Windows PowerShell
. C:/src/highs-go/scripts/enter-dev.ps1
```

```sh
# Linux/macOS: source from Bash or Zsh
source /path/to/highs-go/scripts/enter-dev.sh
# Or reuse an existing installation:
source /path/to/highs-go/scripts/enter-dev.sh --highs-root /path/to/highs
```

These terminal helpers require HiGHS to have been built already. They configure only the current session.

### Manual compilation

Install the same platform prerequisites described above. You can build the pinned native dependency separately without generating a workspace:

```powershell
# Windows PowerShell
./scripts/build-highs.ps1
. ./scripts/enter-dev.ps1
go build ./...
```

```sh
# Linux/macOS, from the repository root
bash ./scripts/build-highs.sh --jobs 4
source ./scripts/enter-dev.sh
go build ./...
```

To compile HiGHS yourself without the build helper, check out the pinned release and configure a shared library with 32-bit `HighsInt`. This Linux/macOS example installs under `$HOME/.local/highs`:

```sh
git clone --depth 1 --branch v1.15.1 https://github.com/ERGO-Code/HiGHS.git highs-src
test "$(git -C highs-src rev-parse HEAD)" = 04024d701f79feb8e2f18bc3df0dffc04ef05088
cmake -S highs-src -B highs-build \
  -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="$HOME/.local/highs" \
  -DCMAKE_INSTALL_LIBDIR=lib -DFAST_BUILD=ON -DBUILD_SHARED_LIBS=ON \
  -DBUILD_SHARED_EXTRAS_LIB=OFF -DBUILD_CXX_EXE=OFF -DBUILD_EXAMPLES=OFF \
  -DBUILD_TESTING=OFF -DZLIB=OFF -DHIPO=OFF -DHIGHSINT64=OFF
cmake --build highs-build --parallel 4
cmake --install highs-build
```

Configure the wrapper's build and runtime paths in your application's terminal:

```sh
export CGO_ENABLED=1
export PKG_CONFIG_PATH="$HOME/.local/highs/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
# Quotes inside the value preserve paths containing spaces when cgo parses flags.
export CGO_LDFLAGS="${CGO_LDFLAGS:+$CGO_LDFLAGS }\"-Wl,-rpath,$HOME/.local/highs/lib\""
go build ./...
```

Standard pkg-config search locations need no `PKG_CONFIG_PATH` override. Use the actual library directory if your installation uses `lib64`. For installations in paths containing spaces, `highs.pc` must quote its `-I` and `-L` paths; the build helper handles this automatically. The runtime search path avoids requiring `LD_LIBRARY_PATH` on Linux or `DYLD_LIBRARY_PATH` on macOS. Windows manual toolchain and compilation details are in [CONTRIBUTING.md](CONTRIBUTING.md#windows-development). The [Dockerfile](Dockerfile) also provides a Linux build and runtime image.

### Giving an application to end users

On Windows, build your application from the configured terminal and bundle it:

```powershell
go build -o app.exe .
& C:/src/highs-go/scripts/bundle-windows.ps1 -Executable ./app.exe -Destination ./dist/my-app
```

Use a new destination directory. The helper copies the executable, its imported non-system DLLs (including transitive dependencies), and HiGHS/Go-wrapper/toolchain notices. Pass the same `-ToolchainBin` and `-HighsRoot` overrides if you used them during setup. Additional application DLLs must be beside the source executable; add your own assets and applicable notices separately. Libraries loaded dynamically at runtime are not detected.

Zip and distribute the **whole folder**. End users extract it and run your executable; they need neither Go, a compiler, nor environment variables. A command-line application's UI remains your application's responsibility. For Linux deployment, the supplied Docker runtime image includes HiGHS and its runtime dependencies.

## Usage

```go
package main

import (
    "fmt"
    "log"
    "math"

    highs "github.com/VladBagat/highs-go"
)

func main() {
    m := highs.Model{Sense: highs.Maximize}
    x := m.AddVariable(3, 0, math.Inf(1))
    y := m.AddVariable(2, 0, math.Inf(1))
    m.AddConstraint(math.Inf(-1), 4,
        highs.Term{Variable: x, Coefficient: 1},
        highs.Term{Variable: y, Coefficient: 1},
    )
    m.AddConstraint(math.Inf(-1), 2, highs.Term{Variable: x, Coefficient: 1})

    result, err := m.Solve()
    if err != nil {
        log.Fatal(err)
    }
    if result.Status != highs.Optimal {
        log.Fatalf("solver status: %s (native %d)", result.Status, result.NativeStatus)
    }
    fmt.Printf("objective=%g x=%g y=%g\n", result.Objective, result.Values[x], result.Values[y])
    // objective=10 x=2 y=2
}
```

### Solve controls and diagnostics

Use `Solve()` for quiet, unlimited solves, or supply options (with `time` and `log` imported):

```go
result, err := m.SolveWithOptions(highs.SolveOptions{
    TimeLimit: 5 * time.Second,
    Log: func(level highs.LogLevel, message string) {
        log.Printf("HiGHS [%d]: %s", level, message)
    },
})
if err != nil {
    return err
}
fmt.Printf("HiGHS %s: %s in %s\n",
    highs.NativeVersion(), result.Status, result.Statistics.RunTime)
```

A zero time limit is unlimited; negative limits are rejected. Limits are cooperative native solver limits, not hard deadlines for the whole Go call. `TimeLimit`, `IterationLimit`, and `Interrupted` are termination statuses, not errors. Solution values remain available only for `Optimal`.

`Statistics.RunTime` excludes Go validation and model packing. Simplex, IPM, crossover and PDLP iteration counts are available when `IterationsAvailable` is true; otherwise all counts are zero. Presolve can finish a model with zero iterations.

Logging is disabled unless `Log` is supplied. Messages retain native text and newlines, with `LogInfo`, `LogDetailed`, `LogVerbose`, `LogWarning`, `LogError`, or `LogUnknown` severity. Callbacks are serialized within each solve, execute synchronously on native callback threads, and must return promptly without re-entering this library. Synchronize callbacks shared between solves yourself. A callback panic suppresses subsequent callbacks and is re-raised with its original value only after native execution and cleanup finish; it does not immediately stop the solve. No callback occurs after the method returns.

## API

| API | Purpose |
| --- | --- |
| `Model{Sense, Offset}` | Select the objective direction and constant offset. The zero value minimizes. |
| `AddVariable(cost, lower, upper)` | Add a continuous variable and return its index. |
| `AddConstraint(lower, upper, terms...)` | Add a bounded linear expression and return its row index. |
| `Term{Variable, Coefficient}` | Specify a variable's coefficient in a constraint. |
| `Solve() (*Result, error)` | Validate the model and solve it with a fresh HiGHS instance. |
| `SolveWithOptions(SolveOptions) (*Result, error)` | Solve with an optional time limit and log callback. |
| `NativeVersion() string` | Report the loaded HiGHS library version. |

Variable and row indices are zero-based and match result ordering. Use `math.Inf(-1)` and `math.Inf(1)` for open bounds. Costs, coefficients, and the objective offset must be finite. Duplicate variables within a constraint and finite bounds with magnitude at least `1e30` are rejected.

`Result.Status` is `Optimal`, `Infeasible`, `Unbounded`, `UnboundedOrInfeasible`, `TimeLimit`, `IterationLimit`, `Interrupted`, or `Other`. `NativeStatus` retains the underlying HiGHS status code. Solution fields—`Objective`, `Values`, `ReducedCosts`, `RowActivities`, and `RowDuals`—are populated only for `Optimal`. Statistics are collected for non-optimal outcomes too.

Invalid input and native API failures return Go errors. Infeasible and unbounded outcomes are reported through status. Check status before reading solution values, and compare floating-point results with tolerances. Model mutation requires synchronization with any concurrent solve.

## Testing and contributing

The test suite covers known LP solutions and duals, objective offsets, maximization, equality constraints, infeasible and unbounded models, empty rows, input validation, and sparse matrix packing.

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, test commands, and local dependency development. CI is configured for Windows x64 with MSYS2 UCRT64, native Linux and macOS setup, and Linux through Docker.

## License

[MIT](LICENSE). HiGHS carries its own MIT license and third-party notices. The supplied builds disable HiPO; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for dependency licensing and redistribution details.
# highs-go
