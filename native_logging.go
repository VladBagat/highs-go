package highs

/*
#include "native_logging.h"
*/
import "C"

import (
	"runtime/cgo"
	"sync"
	"unsafe"
)

type nativeLogger struct {
	mu         sync.Mutex
	log        func(LogLevel, string)
	panicked   bool
	panicValue any
	handle     cgo.Handle
	data       *C.uintptr_t
}

func newNativeLogger(log func(LogLevel, string)) *nativeLogger {
	l := &nativeLogger{log: log}
	l.data = (*C.uintptr_t)(C.malloc(C.size_t(unsafe.Sizeof(C.uintptr_t(0)))))
	l.handle = cgo.NewHandle(l)
	*l.data = C.uintptr_t(l.handle)
	return l
}

// close is called only after Highs_destroy, while the handle is still valid.
func (l *nativeLogger) close() {
	C.free(unsafe.Pointer(l.data))
	l.handle.Delete()
	l.mu.Lock()
	panicked, value := l.panicked, l.panicValue
	l.mu.Unlock()
	if panicked {
		panic(value)
	}
}

// HiGHS 1.15.1's HighsLogType values are a C++ enum, not C API constants.
func nativeLogLevel(level int) LogLevel {
	switch level {
	case 1:
		return LogInfo
	case 2:
		return LogDetailed
	case 3:
		return LogVerbose
	case 4:
		return LogWarning
	case 5:
		return LogError
	default:
		return LogUnknown
	}
}

//export goHighsLog
func goHighsLog(handle C.uintptr_t, level C.int, message *C.char) {
	l := cgo.Handle(handle).Value().(*nativeLogger)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.panicked {
		return
	}
	returned := false
	defer func() {
		if !returned {
			l.panicked = true
			l.panicValue = recover()
		}
	}()
	l.log(nativeLogLevel(int(level)), C.GoString(message))
	returned = true
}
