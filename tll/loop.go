package tll

// #cgo pkg-config: tll
/*
#include <tll/processor/loop.h>
*/
import "C"
import "time"

type Loop struct {
	ptr *C.tll_processor_loop_t
}

func NewLoop(cfg ConstConfig) *Loop {
	if ptr := C.tll_processor_loop_new_cfg(cfg.ptr); ptr != nil {
		return &Loop{ptr}
	}
	return nil
}

func (self *Loop) Free() {
	C.tll_processor_loop_free(self.ptr)
	self.ptr = nil
}

func (self Loop) Add(c Channel) int {
	return int(C.tll_processor_loop_add(self.ptr, c.ptr))
}

func (self Loop) Del(c Channel) int {
	return int(C.tll_processor_loop_del(self.ptr, c.ptr))
}

func (self Loop) Step(timeout time.Duration) int {
	return int(C.tll_processor_loop_step(self.ptr, C.long(timeout.Milliseconds())))
}

func (self Loop) Run(timeout time.Duration) int {
	return int(C.tll_processor_loop_run(self.ptr, C.long(timeout.Milliseconds())))
}

func (self Loop) Stop() int {
	return int(C.tll_processor_loop_stop_get(self.ptr))
}

func (self Loop) SetStop(flag int) int {
	return int(C.tll_processor_loop_stop_set(self.ptr, C.int(flag)))
}
