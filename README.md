# highs-go

A small Go wrapper around [HiGHS](https://highs.dev/) for continuous linear programming.

Built primarily for personal use, with AI-assisted development and AI-generated code. Use with caution and validate results for your application. This is an independent project with an experimental API that may change before v1.0.

## Features

- Continuous variables with lower and upper bounds.
- Minimization and maximization, including an objective offset.
- Equality, one-sided, and ranged linear constraints.
- Solver status, objective value, primal values, reduced costs, row activities, and row duals.

The API covers batch LP solving through the HiGHS C interface. Each solve creates and releases a native solver, with console output disabled. Integer and quadratic programming, solver configuration, callbacks, and warm starts are outside the current scope.

## Installation

```sh
go get github.com/VladBagat/highs-go
```

Requires **Go 1.26+**, **cgo**, a compatible C compiler, and **HiGHS v1.15.1** headers and shared library with 32-bit `HighsInt`. Install the native dependency separately and configure the build and runtime paths for its location.

### Windows x64

Use a MinGW toolchain for both HiGHS and Go. For a HiGHS installation under `C:/highs` and MSYS2 UCRT64 under `C:/msys64`, configure PowerShell:

```powershell
$env:CGO_ENABLED = '1'
$env:CC = 'gcc.exe'
$env:CXX = 'g++.exe'
$env:CGO_CFLAGS = '-IC:/highs/include/highs'
$env:CGO_LDFLAGS = '-LC:/highs/lib'
$env:PATH = "C:\highs\bin;C:\msys64\ucrt64\bin;$env:PATH"
```

HiGHS supplies the headers, `lib/libhighs.dll.a` import library, and DLL in `bin`. The HiGHS and MinGW runtime DLLs must remain available when running the application.

The repository includes scripts to build the pinned HiGHS release and configure a PowerShell session. See [Windows development setup](CONTRIBUTING.md#windows-development) for the source build instructions.

### Linux

For a HiGHS installation under `/opt/highs`:

```sh
export CGO_ENABLED=1
export CGO_CFLAGS='-I/opt/highs/include/highs'
export CGO_LDFLAGS='-L/opt/highs/lib'
export LD_LIBRARY_PATH="/opt/highs/lib${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
```

Adjust paths to match the installation. The [Dockerfile](Dockerfile) provides a complete build using the pinned HiGHS source release.

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

## API

| API | Purpose |
| --- | --- |
| `Model{Sense, Offset}` | Select the objective direction and constant offset. The zero value minimizes. |
| `AddVariable(cost, lower, upper)` | Add a continuous variable and return its index. |
| `AddConstraint(lower, upper, terms...)` | Add a bounded linear expression and return its row index. |
| `Term{Variable, Coefficient}` | Specify a variable's coefficient in a constraint. |
| `Solve() (*Result, error)` | Validate the model and solve it with a fresh HiGHS instance. |

Variable and row indices are zero-based and match result ordering. Use `math.Inf(-1)` and `math.Inf(1)` for open bounds. Costs, coefficients, and the objective offset must be finite. Duplicate variables within a constraint and finite bounds with magnitude at least `1e30` are rejected.

`Result.Status` is `Optimal`, `Infeasible`, `Unbounded`, `UnboundedOrInfeasible`, or `Other`. `NativeStatus` retains the underlying HiGHS status code. Numeric fields—`Objective`, `Values`, `ReducedCosts`, `RowActivities`, and `RowDuals`—are populated only for `Optimal`.

Invalid input and native API failures return Go errors. Infeasible and unbounded outcomes are reported through status. Check status before reading solution values, and compare floating-point results with tolerances. Model mutation requires synchronization with any concurrent solve.

## Testing and contributing

The test suite covers known LP solutions and duals, objective offsets, maximization, equality constraints, infeasible and unbounded models, empty rows, input validation, and sparse matrix packing.

See [CONTRIBUTING.md](CONTRIBUTING.md) for build instructions, test commands, and local dependency development. CI is configured for Windows x64 with MSYS2 UCRT64 and Linux through Docker.

## License

[MIT](LICENSE). HiGHS carries its own MIT license and third-party notices. The supplied builds disable HiPO; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) for dependency licensing and redistribution details.
# highs-go
