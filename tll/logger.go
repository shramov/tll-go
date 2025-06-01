package tll

// #cgo pkg-config: tll
/*
#include <tll/logger.h>
//const char *_GoStringPtr(_GoString_ s);
//static inline const char* _gostr2c(_GoString_ s) { return _GoStringPtr(s); }
*/
import "C"

type Logger struct{ ptr *C.tll_logger_t }

func NewLogger(name string) *Logger {
	ptr := C.tll_logger_new(C._GoStringPtr(name), C.int(len(name)))
	if ptr == nil {
		return nil
	}
	return &Logger{ptr}
}
