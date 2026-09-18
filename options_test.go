package highs

import (
	"math"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testModel() *Model {
	m := &Model{Sense: Maximize}
	x := m.AddVariable(3, 0, 2)
	y := m.AddVariable(2, 0, 3)
	m.AddConstraint(4, 4, Term{x, 1}, Term{y, 1})
	return m
}

func TestNearRejectsNaN(t *testing.T) {
	if os.Getenv("HIGHS_TEST_NAN") == "1" {
		near(t, math.NaN(), 1)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNearRejectsNaN$")
	cmd.Env = append(os.Environ(), "HIGHS_TEST_NAN=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "got NaN") {
		t.Fatalf("NaN accepted or unexpected failure: %v %s", err, out)
	}
}

func TestZeroSolveOptions(t *testing.T) {
	m := testModel()
	a, err := m.Solve()
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.SolveWithOptions(SolveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	a.Statistics = SolveStatistics{}
	b.Statistics = SolveStatistics{}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("different results: %+v %+v", a, b)
	}
}

func TestStatisticsAndVersion(t *testing.T) {
	for _, infeasible := range []bool{false, true} {
		m := testModel()
		if infeasible {
			m.AddConstraint(10, 11, Term{0, 1})
		}
		r, err := m.Solve()
		if err != nil {
			t.Fatal(err)
		}
		s := r.Statistics
		if s.RunTime < 0 {
			t.Fatalf("negative runtime: %+v", s)
		}
		if !s.IterationsAvailable {
			t.Fatalf("missing iteration info: %+v", s)
		}
		if s.SimplexIterations < 0 || s.IPMIterations < 0 || s.CrossoverIterations < 0 || s.PDLPIterations < 0 {
			t.Fatalf("negative iterations: %+v", s)
		}
	}
	version := NativeVersion()
	if version == "" {
		t.Fatal("empty native version")
	}
	if want := os.Getenv("HIGHS_EXPECT_VERSION"); want != "" && version != want {
		t.Fatalf("version %s, want %s", version, want)
	}
}

func TestNegativeTimeLimit(t *testing.T) {
	r, err := testModel().SolveWithOptions(SolveOptions{TimeLimit: -time.Second})
	if r != nil || err == nil || !strings.Contains(err.Error(), "time limit") {
		t.Fatalf("got %v, %v", r, err)
	}
}

func TestTimeLimit(t *testing.T) {
	m := &Model{Sense: Maximize}
	const n = 200
	for j := 0; j < n; j++ {
		m.AddVariable(float64(j%17+1), 0, 1)
	}
	for i := 0; i < n; i++ {
		terms := make([]Term, n)
		for j := range terms {
			terms[j] = Term{j, float64((i*37+j*13+i*j)%101 + 1)}
		}
		m.AddConstraint(math.Inf(-1), 2000, terms...)
	}
	r, err := m.SolveWithOptions(SolveOptions{TimeLimit: time.Nanosecond})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != TimeLimit {
		t.Fatalf("status = %s", r.Status)
	}
	if s := r.Statistics; s.RunTime < 0 || (!s.IterationsAvailable &&
		(s.SimplexIterations != 0 || s.IPMIterations != 0 || s.CrossoverIterations != 0 || s.PDLPIterations != 0)) {
		t.Fatalf("invalid time-limited statistics: %+v", s)
	}
	if r.Objective != 0 || len(r.Values)+len(r.ReducedCosts)+len(r.RowActivities)+len(r.RowDuals) != 0 {
		t.Fatalf("unexpected solution: %+v", r)
	}
}

func TestTerminationStatuses(t *testing.T) {
	for raw, want := range map[int]Status{13: TimeLimit, 14: IterationLimit, 17: Interrupted, -999: Other} {
		if got := nativeStatus(raw); got != want {
			t.Fatalf("%d: got %s, want %s", raw, got, want)
		}
	}
}
