package highs

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestNativeOutput(t *testing.T) {
	if mode := os.Getenv("HIGHS_TEST_OUTPUT"); mode != "" {
		opts := SolveOptions{}
		calls := 0
		if mode == "log" {
			opts.Log = func(level LogLevel, message string) {
				calls++
				if message == "" {
					t.Error("empty message")
				}
				if level == LogUnknown {
					t.Error("unexpected unknown level")
				}
			}
		}
		if _, err := testModel().SolveWithOptions(opts); err != nil {
			t.Fatal(err)
		}
		if mode == "log" && calls == 0 {
			t.Fatal("no log callbacks")
		}
		return
	}
	for _, mode := range []string{"silent", "log"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestNativeOutput$")
		cmd.Env = append(os.Environ(), "HIGHS_TEST_OUTPUT="+mode)
		out, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "PASS" {
			t.Fatalf("%s: %v, output %q", mode, err, out)
		}
	}
}

func TestLoggingLifetime(t *testing.T) {
	calls := [2]int{}
	for i := range calls {
		_, err := testModel().SolveWithOptions(SolveOptions{Log: func(_ LogLevel, _ string) { calls[i]++ }})
		if err != nil {
			t.Fatal(err)
		}
		if calls[i] == 0 {
			t.Fatal("no callbacks")
		}
	}
	before := calls
	if _, err := testModel().Solve(); err != nil {
		t.Fatal(err)
	}
	if calls != before {
		t.Fatal("callback used after its solve")
	}
}

func TestLoggingPanic(t *testing.T) {
	token := &struct{}{}
	calls := 0
	func() {
		defer func() {
			if got := recover(); got != token {
				t.Errorf("panic = %v, want original token", got)
			}
		}()
		_, _ = testModel().SolveWithOptions(SolveOptions{Log: func(_ LogLevel, _ string) { calls++; panic(token) }})
		t.Error("callback panic was swallowed")
	}()
	if calls != 1 {
		t.Fatalf("callback invoked %d times after panic", calls)
	}
	if _, err := testModel().Solve(); err != nil {
		t.Fatal(err)
	}
}

func TestLogLevels(t *testing.T) {
	for raw, want := range map[int]LogLevel{1: LogInfo, 2: LogDetailed, 3: LogVerbose, 4: LogWarning, 5: LogError, -1: LogUnknown} {
		if got := nativeLogLevel(raw); got != want {
			t.Fatalf("level %d = %v, want %v", raw, got, want)
		}
	}
}
