package main

import (
	"fmt"
	"os"
	"time"

	highs "github.com/VladBagat/highs-go"
)

func main() {
	fmt.Printf("HiGHS version=%s\n", highs.NativeVersion())
	model := highs.Model{Sense: highs.Maximize, Offset: 0}

	m := model.AddVariable(0, 0, 4)
	f := model.AddVariable(1, 0, 3)

	model.AddConstraint(0, 0, highs.Term{Variable: m, Coefficient: 1}, highs.Term{Variable: f, Coefficient: -2})

	levels := map[highs.LogLevel]string{
		highs.LogUnknown: "unknown", highs.LogInfo: "info",
		highs.LogDetailed: "detailed", highs.LogVerbose: "verbose",
		highs.LogWarning: "warning", highs.LogError: "error",
	}
	r, err := model.SolveWithOptions(highs.SolveOptions{
		// A cooperative solver limit; zero would mean unlimited.
		TimeLimit: 5 * time.Second,
		// Omit Log to keep the solver silent. Native messages include newlines.
		Log: func(level highs.LogLevel, message string) {
			fmt.Fprintf(os.Stderr, "[HiGHS %s] %s", levels[level], message)
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("status=%s native_status=%d\n", r.Status, r.NativeStatus)
	s := r.Statistics
	fmt.Printf("solver_time=%s\n", s.RunTime)
	if s.IterationsAvailable {
		// Presolve can solve this small model with zero iterations.
		fmt.Printf("iterations: simplex=%d ipm=%d crossover=%d pdlp=%d\n",
			s.SimplexIterations, s.IPMIterations, s.CrossoverIterations, s.PDLPIterations)
	} else {
		fmt.Println("iteration statistics unavailable")
	}

	// Limits and interruption are outcomes, not Go errors. Only Optimal has values.
	switch r.Status {
	case highs.Optimal:
		fmt.Printf("objective=%.6g x=%.6g y=%.6g\n", r.Objective, r.Values[m], r.Values[f])
	case highs.TimeLimit, highs.IterationLimit, highs.Interrupted:
		fmt.Println("Solve stopped early; no solution values are exposed.")
	case highs.Infeasible:
		fmt.Println("The constraints have no feasible solution.")
	case highs.Unbounded, highs.UnboundedOrInfeasible:
		fmt.Println("No finite optimum was established; check the model bounds and constraints.")
	default:
		fmt.Println("See the native status and log messages for details.")
	}
}
