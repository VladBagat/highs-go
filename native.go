package highs

/*
#cgo windows LDFLAGS: -lhighs -lstdc++
#cgo !windows pkg-config: highs
#include <stdlib.h>
#include "interfaces/highs_c_api.h"
#include "native_logging.h"
// Keep the full C API header in this translation unit: HiGHS 1.15.1 also
// defines some externally linked constants that cannot be included twice.
static inline HighsInt highs_go_enable_logging(void* highs, uintptr_t* handle) {
    HighsInt status = Highs_setCallback(highs, highs_go_log, handle);
    if (status != kHighsStatusOk) return status;
    return Highs_startCallback(highs, kHighsCallbackLogging);
}
// HiGHS declares these as static const variables; cgo cannot link to them
// directly from its generated translation units.
static inline HighsInt go_status_ok(void) { return kHighsStatusOk; }
static inline HighsInt go_status_error(void) { return kHighsStatusError; }
static inline HighsInt go_status_warning(void) { return kHighsStatusWarning; }
static inline HighsInt go_sense_min(void) { return kHighsObjSenseMinimize; }
static inline HighsInt go_sense_max(void) { return kHighsObjSenseMaximize; }
static inline HighsInt go_matrix_rowwise(void) { return kHighsMatrixFormatRowwise; }
static inline HighsInt go_model_optimal(void) { return kHighsModelStatusOptimal; }
static inline HighsInt go_model_infeasible(void) { return kHighsModelStatusInfeasible; }
static inline HighsInt go_model_unbounded(void) { return kHighsModelStatusUnbounded; }
static inline HighsInt go_model_ambiguous(void) { return kHighsModelStatusUnboundedOrInfeasible; }
static inline HighsInt go_model_time_limit(void) { return kHighsModelStatusTimeLimit; }
static inline HighsInt go_model_iteration_limit(void) { return kHighsModelStatusIterationLimit; }
static inline HighsInt go_model_interrupt(void) { return kHighsModelStatusInterrupt; }
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

// NativeVersion returns the version of the loaded HiGHS library.
func NativeVersion() string {
	return C.GoString(C.Highs_version())
}

func nativeStatistics(h unsafe.Pointer) (SolveStatistics, error) {
	s := SolveStatistics{RunTime: time.Duration(float64(C.Highs_getRunTime(h)) * float64(time.Second))}
	counts := [4]int{}
	available := true
	for i, key := range [...]string{"simplex_iteration_count", "ipm_iteration_count", "crossover_iteration_count", "pdlp_iteration_count"} {
		name := C.CString(key)
		var count C.HighsInt
		code := C.Highs_getIntInfoValue(h, name, &count)
		C.free(unsafe.Pointer(name))
		switch code {
		case C.go_status_ok():
			counts[i] = int(count)
		case C.go_status_warning():
			available = false
		default:
			return s, fmt.Errorf("HiGHS could not read %s: status %d", key, int(code))
		}
	}
	if available {
		s.IterationsAvailable = true
		s.SimplexIterations, s.IPMIterations, s.CrossoverIterations, s.PDLPIterations = counts[0], counts[1], counts[2], counts[3]
	}
	return s, nil
}

func doubles(v []float64) *C.double {
	if len(v) == 0 {
		return nil
	}
	return (*C.double)(unsafe.Pointer(&v[0]))
}

func ints(v []C.HighsInt) *C.HighsInt {
	if len(v) == 0 {
		return nil
	}
	return &v[0]
}

func nativeInts(v []int) []C.HighsInt {
	result := make([]C.HighsInt, len(v))
	for i, n := range v {
		result[i] = C.HighsInt(n)
	}
	return result
}

func nativeStatus(s int) Status {
	switch C.HighsInt(s) {
	case C.go_model_optimal():
		return Optimal
	case C.go_model_infeasible():
		return Infeasible
	case C.go_model_unbounded():
		return Unbounded
	case C.go_model_ambiguous():
		return UnboundedOrInfeasible
	case C.go_model_time_limit():
		return TimeLimit
	case C.go_model_iteration_limit():
		return IterationLimit
	case C.go_model_interrupt():
		return Interrupted
	default:
		return Other
	}
}

func solveNative(sense Sense, offset float64, p packedModel, opts SolveOptions) (*Result, error) {
	h := C.Highs_create()
	if h == nil {
		return nil, fmt.Errorf("HiGHS could not create a solver")
	}
	var logger *nativeLogger
	defer func() {
		C.Highs_destroy(h)
		if logger != nil {
			logger.close()
		}
	}()

	// The wrapper returns data rather than writing solver output to stdout.
	if err := setNativeBool(h, "output_flag", false); err != nil {
		return nil, err
	}
	if err := setNativeBool(h, "log_to_console", false); err != nil {
		return nil, err
	}
	if opts.Log != nil {
		logger = newNativeLogger(opts.Log)
		if code := C.highs_go_enable_logging(h, logger.data); code != C.go_status_ok() {
			return nil, fmt.Errorf("HiGHS could not enable logging: status %d", int(code))
		}
		if err := setNativeBool(h, "output_flag", true); err != nil {
			return nil, err
		}
	}
	if opts.TimeLimit > 0 {
		name := C.CString("time_limit")
		code := C.Highs_setDoubleOptionValue(h, name, C.double(opts.TimeLimit.Seconds()))
		C.free(unsafe.Pointer(name))
		if code != C.go_status_ok() {
			return nil, fmt.Errorf("HiGHS could not set time limit: status %d", int(code))
		}
	}

	starts := nativeInts(p.start)
	indices := nativeInts(p.index)
	csense := C.go_sense_min()
	if sense == Maximize {
		csense = C.go_sense_max()
	}
	code := C.Highs_passLp(h,
		C.HighsInt(len(p.cost)), C.HighsInt(len(p.rowLower)), C.HighsInt(len(p.index)),
		C.go_matrix_rowwise(), csense, C.double(offset),
		doubles(p.cost), doubles(p.colLower), doubles(p.colUpper),
		doubles(p.rowLower), doubles(p.rowUpper), ints(starts), ints(indices), doubles(p.value))
	if code != C.go_status_ok() {
		return nil, fmt.Errorf("HiGHS rejected the LP: status %d", int(code))
	}
	if code = C.Highs_run(h); code == C.go_status_error() {
		return nil, fmt.Errorf("HiGHS solve failed: status %d", int(code))
	}
	raw := C.Highs_getModelStatus(h)
	r := &Result{Status: nativeStatus(int(raw)), NativeStatus: int(raw)}
	var err error
	r.Statistics, err = nativeStatistics(h)
	if err != nil {
		return nil, err
	}
	if r.Status != Optimal {
		return r, nil
	}
	r.Objective = float64(C.Highs_getObjectiveValue(h))
	r.Values = make([]float64, len(p.cost))
	r.ReducedCosts = make([]float64, len(p.cost))
	r.RowActivities = make([]float64, len(p.rowLower))
	r.RowDuals = make([]float64, len(p.rowLower))
	code = C.Highs_getSolution(h, doubles(r.Values), doubles(r.ReducedCosts),
		doubles(r.RowActivities), doubles(r.RowDuals))
	if code != C.go_status_ok() {
		return nil, fmt.Errorf("HiGHS could not read the solution: status %d", int(code))
	}
	return r, nil
}

func setNativeBool(h unsafe.Pointer, key string, enabled bool) error {
	name := C.CString(key)
	defer C.free(unsafe.Pointer(name))
	var value C.HighsInt
	if enabled {
		value = 1
	}
	if code := C.Highs_setBoolOptionValue(h, name, value); code != C.go_status_ok() {
		return fmt.Errorf("HiGHS could not set %s: status %d", key, int(code))
	}
	return nil
}
