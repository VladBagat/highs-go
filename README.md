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

### Windows x64

Install [Go](https://go.dev/dl/), Git, CMake and [MSYS2](https://www.msys2.org/). In the MSYS2 **UCRT64** terminal, install the compiler once:

```sh
pacman -S --needed mingw-w64-ucrt-x86_64-gcc mingw-w64-ucrt-x86_64-make
```

Keep a checkout of this repository for the native dependencies and helpers. From your **application's directory** in PowerShell, run:

```powershell
& C:/src/highs-go/scripts/setup.ps1
code ./highs-go.code-workspace
```

Replace `C:/src/highs-go` with your checkout. Setup builds HiGHS if needed, checks a real solve, and configures both VS Code's Go tools and new integrated terminals. Open this workspace on subsequent visits; no repeated environment commands are needed. Add `highs-go.code-workspace` to your application's `.gitignore` because it contains local paths.

Compiler discovery uses `PATH`, then `C:/msys64/ucrt64/bin`. Use `-ToolchainBin C:/path/to/mingw/bin` to override it, or `-HighsRoot C:/highs` for an existing compatible installation. Rerun setup if you move either installation. For another editor or an external terminal, use `. C:/src/highs-go/scripts/enter-dev.ps1` to configure that session.

### Linux / macOS

Install a C compiler, `pkg-config`, and compatible HiGHS headers/shared libraries. The wrapper discovers build flags through HiGHS's `highs.pc`. For a custom installation under `/opt/highs`:

```sh
export CGO_ENABLED=1
export PKG_CONFIG_PATH="/opt/highs/lib/pkgconfig${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
```

Standard pkg-config search locations need no override. The shared library must also be discoverable by the OS loader; for a custom Linux installation, set `LD_LIBRARY_PATH=/opt/highs/lib` or configure the loader. The [Dockerfile](Dockerfile) provides a tested Linux build and runtime image. macOS discovery is supported through pkg-config but is not currently tested in CI.

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

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, test commands, and local dependency development. CI is configured for Windows x64 with MSYS2 UCRT64 and Linux through Docker.

## License

[MIT](LICENSE). HiGHS carries its own MIT license and third-party notices. The supplied builds disable HiPO; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for dependency licensing and redistribution details.
# highs-go
