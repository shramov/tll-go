package tll

// #cgo pkg-config: tll
/*
#include <tll/logger.h>
//const char *_GoStringPtr(_GoString_ s);
//static inline const char* _gostr2c(_GoString_ s) { return _GoStringPtr(s); }
*/
import "C"
import "fmt"

type Logger struct{ ptr *C.tll_logger_t }

type LoggerLevel int

const (
	LoggerTrace    LoggerLevel = C.TLL_LOGGER_TRACE
	LoggerDebug    LoggerLevel = C.TLL_LOGGER_DEBUG
	LoggerInfo     LoggerLevel = C.TLL_LOGGER_INFO
	LoggerWarning  LoggerLevel = C.TLL_LOGGER_WARNING
	LoggerError    LoggerLevel = C.TLL_LOGGER_ERROR
	LoggerCritical LoggerLevel = C.TLL_LOGGER_CRITICAL
)

func NewLogger(name string) *Logger {
	ptr := C.tll_logger_new(C._GoStringPtr(name), C.int(len(name)))
	if ptr == nil {
		return nil
	}
	return &Logger{ptr}
}

func LoggerConfig(cfg ConstConfig) {
	C.tll_logger_config(cfg.ptr)
}

func LoggerConfigMap(settings map[string]string) {
	c := ConfigFromMap(settings)
	defer c.Free()
	C.tll_logger_config(c.ptr)
}

func (self Logger) Log(level LoggerLevel, s string) {
	if level < LoggerLevel(self.ptr.level) {
		return
	}
	C.tll_logger_log(self.ptr, C.tll_logger_level_t(level), C._GoStringPtr(s), C.size_t(len(s)))
}

func (self Logger) Logf(level LoggerLevel, format string, a ...any) {
	if level < LoggerLevel(self.ptr.level) {
		return
	}
	s := fmt.Sprintf(format, a...)
	C.tll_logger_log(self.ptr, C.tll_logger_level_t(level), C._GoStringPtr(s), C.size_t(len(s)))
}

func (self Logger) Trace(s string)    { self.Log(LoggerTrace, s) }
func (self Logger) Debug(s string)    { self.Log(LoggerDebug, s) }
func (self Logger) Info(s string)     { self.Log(LoggerInfo, s) }
func (self Logger) Warning(s string)  { self.Log(LoggerWarning, s) }
func (self Logger) Error(s string)    { self.Log(LoggerError, s) }
func (self Logger) Critical(s string) { self.Log(LoggerCritical, s) }

func (self Logger) Tracef(format string, a ...any)    { self.Logf(LoggerTrace, format, a...) }
func (self Logger) Debugf(format string, a ...any)    { self.Logf(LoggerDebug, format, a...) }
func (self Logger) Infof(format string, a ...any)     { self.Logf(LoggerInfo, format, a...) }
func (self Logger) Warningf(format string, a ...any)  { self.Logf(LoggerWarning, format, a...) }
func (self Logger) Errorf(format string, a ...any)    { self.Logf(LoggerError, format, a...) }
func (self Logger) Criticalf(format string, a ...any) { self.Logf(LoggerCritical, format, a...) }
