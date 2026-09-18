package highs

import (
	"math"
	"strings"
	"testing"
)

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-7 {
		t.Fatalf("got %.12g, want %.12g", got, want)
	}
}

// Adapted from HiGHS/examples/call_highs_from_c.c, minimal_api.
func TestPublishedLP(t *testing.T) {
	m := Model{Sense: Minimize, Offset: 3}
	x := m.AddVariable(1, 0, 4)
	y := m.AddVariable(1, 1, math.Inf(1))
	m.AddConstraint(math.Inf(-1), 7, Term{y, 1})
	m.AddConstraint(5, 15, Term{x, 1}, Term{y, 2})
	m.AddConstraint(6, math.Inf(1), Term{x, 3}, Term{y, 2})

	r, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Optimal {
		t.Fatalf("status = %q, want optimal", r.Status)
	}
	near(t, r.Objective, 5.75)
	near(t, r.Values[x], 0.5)
	near(t, r.Values[y], 2.25)
	near(t, r.RowActivities[0], 2.25)
	near(t, r.RowActivities[1], 5)
	near(t, r.RowActivities[2], 6)
	if len(r.ReducedCosts) != 2 || len(r.RowDuals) != 3 {
		t.Fatalf("wrong dual dimensions: cols=%d rows=%d", len(r.ReducedCosts), len(r.RowDuals))
	}
	near(t, r.ReducedCosts[x], 0)
	near(t, r.ReducedCosts[y], 0)
	near(t, r.RowDuals[0], 0)
	near(t, r.RowDuals[1], 0.25)
	near(t, r.RowDuals[2], 0.25)
}

func TestMaximizeAndEquality(t *testing.T) {
	m := Model{Sense: Maximize}
	x := m.AddVariable(3, 0, math.Inf(1))
	y := m.AddVariable(2, 0, math.Inf(1))
	m.AddConstraint(4, 4, Term{x, 1}, Term{y, 1})
	m.AddConstraint(math.Inf(-1), 2, Term{x, 1})
	m.AddConstraint(math.Inf(-1), 3, Term{y, 1})
	r, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Optimal {
		t.Fatalf("status = %q, want optimal", r.Status)
	}
	near(t, r.Objective, 10)
	near(t, r.Values[x], 2)
	near(t, r.Values[y], 2)
	near(t, r.RowActivities[0], 4)
}

func TestInfeasibleIsStatus(t *testing.T) {
	m := Model{Sense: Minimize}
	x := m.AddVariable(1, 0, 1)
	m.AddConstraint(2, math.Inf(1), Term{x, 1})
	r, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Infeasible {
		t.Fatalf("status = %q, want infeasible", r.Status)
	}
	if len(r.Values) != 0 {
		t.Fatalf("infeasible result returned primal values: %v", r.Values)
	}
}

func TestUnboundedIsStatus(t *testing.T) {
	m := Model{Sense: Minimize}
	m.AddVariable(-1, 0, math.Inf(1))
	r, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Unbounded {
		t.Fatalf("status = %q, want unbounded", r.Status)
	}
}

func TestInvalidModels(t *testing.T) {
	tests := []struct {
		name string
		make func() Model
		want string
	}{
		{"bad sense", func() Model { m := Model{Sense: Sense(99)}; m.AddVariable(1, 0, 1); return m }, "sense"},
		{"bad offset", func() Model { m := Model{Offset: math.NaN()}; m.AddVariable(1, 0, 1); return m }, "offset"},
		{"reversed variable bounds", func() Model { m := Model{}; m.AddVariable(1, 2, 1); return m }, "variable 0"},
		{"infinite cost", func() Model { m := Model{}; m.AddVariable(math.Inf(1), 0, 1); return m }, "cost"},
		{"wrong infinity side", func() Model { m := Model{}; m.AddVariable(1, math.Inf(1), math.Inf(1)); return m }, "variable 0"},
		{"reversed row bounds", func() Model { m := Model{}; m.AddVariable(1, 0, 1); m.AddConstraint(2, 1); return m }, "constraint 0"},
		{"invalid index", func() Model { m := Model{}; m.AddVariable(1, 0, 1); m.AddConstraint(0, 1, Term{1, 2}); return m }, "index"},
		{"nan coefficient", func() Model {
			m := Model{}
			m.AddVariable(1, 0, 1)
			m.AddConstraint(0, 1, Term{0, math.NaN()})
			return m
		}, "coefficient"},
		{"duplicate term", func() Model {
			m := Model{}
			m.AddVariable(1, 0, 1)
			m.AddConstraint(0, 1, Term{0, 1}, Term{0, 2})
			return m
		}, "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.make()
			_, err := m.Solve()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestEmptyCoefficientRow(t *testing.T) {
	m := Model{}
	x := m.AddVariable(1, 0, 4)
	m.AddConstraint(math.Inf(-1), 10)
	m.AddConstraint(2, math.Inf(1), Term{x, 1})
	r, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != Optimal {
		t.Fatalf("status = %q, want optimal", r.Status)
	}
	near(t, r.Values[x], 2)
	near(t, r.RowActivities[0], 0)
}

func TestSparseRowsArePackedWithEmptyRow(t *testing.T) {
	m := Model{}
	x := m.AddVariable(2, 0, 5)
	y := m.AddVariable(3, 0, 5)
	m.AddConstraint(0, 4, Term{y, 7}, Term{x, 2})
	m.AddConstraint(math.Inf(-1), 9)
	m.AddConstraint(1, math.Inf(1), Term{x, -3})
	p, err := m.pack(1e30)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := []int{0, 2, 2}
	wantIndex := []int{1, 0, 0}
	wantValue := []float64{7, 2, -3}
	for i, want := range wantStart {
		if p.start[i] != want {
			t.Fatalf("start[%d] = %d, want %d", i, p.start[i], want)
		}
	}
	for i, want := range wantIndex {
		if p.index[i] != want || p.value[i] != wantValue[i] {
			t.Fatalf("entry[%d] = (%d,%g), want (%d,%g)", i, p.index[i], p.value[i], want, wantValue[i])
		}
	}
	if p.rowLower[1] != -1e30 || p.rowUpper[2] != 1e30 {
		t.Fatalf("infinite bounds not translated: %v %v", p.rowLower, p.rowUpper)
	}
}
