package highs

/*
#cgo windows LDFLAGS: -lhighs -lstdc++
#cgo !windows pkg-config: highs
#include <stdlib.h>
#include "interfaces/highs_c_api.h"
// HiGHS declares these as static const variables; cgo cannot link to them
// directly from its generated translation units.
static inline HighsInt go_status_ok(void) { return kHighsStatusOk; }
static inline HighsInt go_status_error(void) { return kHighsStatusError; }
static inline HighsInt go_sense_min(void) { return kHighsObjSenseMinimize; }
static inline HighsInt go_sense_max(void) { return kHighsObjSenseMaximize; }
static inline HighsInt go_matrix_rowwise(void) { return kHighsMatrixFormatRowwise; }
static inline HighsInt go_model_optimal(void) { return kHighsModelStatusOptimal; }
static inline HighsInt go_model_infeasible(void) { return kHighsModelStatusInfeasible; }
static inline HighsInt go_model_unbounded(void) { return kHighsModelStatusUnbounded; }
static inline HighsInt go_model_ambiguous(void) { return kHighsModelStatusUnboundedOrInfeasible; }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

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

func nativeStatus(s C.HighsInt) Status {
	switch s {
	case C.go_model_optimal():
		return Optimal
	case C.go_model_infeasible():
		return Infeasible
	case C.go_model_unbounded():
		return Unbounded
	case C.go_model_ambiguous():
		return UnboundedOrInfeasible
	default:
		return Other
	}
}

func solveNative(sense Sense, offset float64, p packedModel) (*Result, error) {
	h := C.Highs_create()
	if h == nil {
		return nil, fmt.Errorf("HiGHS could not create a solver")
	}
	defer C.Highs_destroy(h)

	// The wrapper returns data rather than writing solver output to stdout.
	option := C.CString("output_flag")
	defer C.free(unsafe.Pointer(option))
	if code := C.Highs_setBoolOptionValue(h, option, 0); code != C.go_status_ok() {
		return nil, fmt.Errorf("HiGHS could not disable output: status %d", int(code))
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
	r := &Result{Status: nativeStatus(raw), NativeStatus: int(raw)}
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
