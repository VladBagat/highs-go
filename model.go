// Package highs solves continuous linear programs with the HiGHS C API.
// A Model holds Go data; each Solve call creates and releases its own native solver.
package highs

import (
	"fmt"
	"math"
	"time"
)

// Sense selects the direction of optimization. The zero value minimizes.
type Sense uint8

const (
	Minimize Sense = iota
	Maximize
)

// Term is a coefficient for a variable in a linear constraint.
type Term struct {
	Variable    int
	Coefficient float64
}

type variable struct{ cost, lower, upper float64 }
type constraint struct {
	lower, upper float64
	terms        []Term
}

// Model represents min/max Offset + sum(cost[i] * x[i]), subject to
// variable and linear constraint bounds. Use math.Inf(-1) and math.Inf(1)
// for open lower and upper bounds respectively. A zero-value Model minimizes.
type Model struct {
	Sense  Sense
	Offset float64
	vars   []variable
	rows   []constraint
}

// AddVariable appends a continuous variable and returns its zero-based index.
// Input is validated by Solve so a model can be assembled without errors.
func (m *Model) AddVariable(cost, lower, upper float64) int {
	i := len(m.vars)
	m.vars = append(m.vars, variable{cost, lower, upper})
	return i
}

// AddConstraint appends lower <= sum(terms) <= upper and returns its row index.
// Terms are copied, so later changes to the caller's slice do not change the model.
func (m *Model) AddConstraint(lower, upper float64, terms ...Term) int {
	i := len(m.rows)
	m.rows = append(m.rows, constraint{lower, upper, append([]Term(nil), terms...)})
	return i
}

// Status is the optimization outcome. Other covers any additional native status.
type Status string

const (
	Optimal               Status = "optimal"
	Infeasible            Status = "infeasible"
	Unbounded             Status = "unbounded"
	UnboundedOrInfeasible Status = "unbounded_or_infeasible"
	Other                 Status = "other"
	TimeLimit             Status = "time_limit"
	IterationLimit        Status = "iteration_limit"
	Interrupted           Status = "interrupted"
)

// Result contains a solver status and statistics. Solution values are populated only for
// Optimal; their indices match the order of AddVariable and AddConstraint calls.
type Result struct {
	Status        Status
	NativeStatus  int
	Objective     float64
	Values        []float64
	ReducedCosts  []float64
	RowActivities []float64
	RowDuals      []float64
	Statistics    SolveStatistics
}

// SolveStatistics describes native solver work, excluding Go validation and packing.
type SolveStatistics struct {
	RunTime time.Duration
	// IterationsAvailable is false when native iteration information is unavailable.
	// In that case all iteration counts are zero.
	IterationsAvailable bool
	SimplexIterations   int
	IPMIterations       int
	CrossoverIterations int
	PDLPIterations      int
}

// Solve validates the model, solves it with a fresh HiGHS instance, and returns
// a result. Infeasible and unbounded models return a status, not an error.
func (m *Model) Solve() (*Result, error) {
	return m.SolveWithOptions(SolveOptions{})
}

// SolveOptions controls a single solve. Its zero value preserves Solve defaults.
type SolveOptions struct {
	// TimeLimit limits native solver time, not the whole Go call. Zero is unlimited.
	TimeLimit time.Duration
	// Log receives native messages, including their original newlines. Nil is silent.
	// Calls are serialized within a solve and run synchronously on native callback
	// threads. The callback must not re-enter this library and should return promptly.
	// A callback shared between solves must provide its own synchronization.
	// Panics are re-raised after native execution and cleanup have completed.
	Log func(LogLevel, string)
}

// LogLevel identifies the severity of a native log message.
type LogLevel uint8

const (
	LogUnknown LogLevel = iota
	LogInfo
	LogDetailed
	LogVerbose
	LogWarning
	LogError
)

// SolveWithOptions validates and solves the model with a fresh native solver.
// Reaching a solver limit returns a result status, not an error.
func (m *Model) SolveWithOptions(opts SolveOptions) (*Result, error) {
	if m == nil {
		return nil, fmt.Errorf("nil model")
	}
	if opts.TimeLimit < 0 {
		return nil, fmt.Errorf("time limit must not be negative")
	}
	p, err := m.pack(1e30) // HiGHS v1.15.1's finite infinity threshold.
	if err != nil {
		return nil, err
	}
	return solveNative(m.Sense, m.Offset, p, opts)
}

type packedModel struct {
	cost, colLower, colUpper []float64
	rowLower, rowUpper       []float64
	start, index             []int
	value                    []float64
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func bound(x, infinity float64, lower bool) (float64, error) {
	if math.IsNaN(x) || (lower && math.IsInf(x, 1)) || (!lower && math.IsInf(x, -1)) {
		return 0, fmt.Errorf("invalid bound %g", x)
	}
	if math.IsInf(x, -1) {
		return -infinity, nil
	}
	if math.IsInf(x, 1) {
		return infinity, nil
	}
	if math.Abs(x) >= infinity {
		return 0, fmt.Errorf("bound %g reaches HiGHS infinity %g", x, infinity)
	}
	return x, nil
}

func (m *Model) pack(infinity float64) (packedModel, error) {
	var p packedModel
	if m.Sense != Minimize && m.Sense != Maximize {
		return p, fmt.Errorf("invalid objective sense %d", m.Sense)
	}
	if !finite(m.Offset) {
		return p, fmt.Errorf("invalid objective offset %g", m.Offset)
	}
	if len(m.vars) == 0 {
		return p, fmt.Errorf("model has no variables")
	}
	// The Docker build uses the standard 32-bit HighsInt configuration.
	if len(m.vars) > math.MaxInt32 || len(m.rows) > math.MaxInt32 {
		return p, fmt.Errorf("model dimensions exceed HighsInt range")
	}
	for i, v := range m.vars {
		if !finite(v.cost) {
			return p, fmt.Errorf("variable %d: invalid cost %g", i, v.cost)
		}
		lo, err := bound(v.lower, infinity, true)
		if err != nil {
			return p, fmt.Errorf("variable %d: %w", i, err)
		}
		hi, err := bound(v.upper, infinity, false)
		if err != nil {
			return p, fmt.Errorf("variable %d: %w", i, err)
		}
		if lo > hi {
			return p, fmt.Errorf("variable %d: lower bound exceeds upper bound", i)
		}
		p.cost = append(p.cost, v.cost)
		p.colLower = append(p.colLower, lo)
		p.colUpper = append(p.colUpper, hi)
	}
	for i, row := range m.rows {
		lo, err := bound(row.lower, infinity, true)
		if err != nil {
			return p, fmt.Errorf("constraint %d: %w", i, err)
		}
		hi, err := bound(row.upper, infinity, false)
		if err != nil {
			return p, fmt.Errorf("constraint %d: %w", i, err)
		}
		if lo > hi {
			return p, fmt.Errorf("constraint %d: lower bound exceeds upper bound", i)
		}
		p.rowLower = append(p.rowLower, lo)
		p.rowUpper = append(p.rowUpper, hi)
		p.start = append(p.start, len(p.index))
		seen := make(map[int]struct{}, len(row.terms))
		for _, term := range row.terms {
			if term.Variable < 0 || term.Variable >= len(m.vars) {
				return p, fmt.Errorf("constraint %d: variable index %d is out of range", i, term.Variable)
			}
			if !finite(term.Coefficient) {
				return p, fmt.Errorf("constraint %d: invalid coefficient %g", i, term.Coefficient)
			}
			if _, exists := seen[term.Variable]; exists {
				return p, fmt.Errorf("constraint %d: duplicate variable %d", i, term.Variable)
			}
			seen[term.Variable] = struct{}{}
			if term.Coefficient != 0 {
				p.index = append(p.index, term.Variable)
				p.value = append(p.value, term.Coefficient)
			}
		}
		if len(p.index) > math.MaxInt32 {
			return p, fmt.Errorf("nonzero count exceeds HighsInt range")
		}
	}
	return p, nil
}
