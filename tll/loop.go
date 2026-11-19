package tll

// #cgo pkg-config: tll
/*
#include <tll/processor/loop.h>
*/
import "C"
import "time"

// Loop drives channels: a channel makes progress (connects, reads, runs callbacks) only while
// the Loop is stepped. A channel added to a Loop stays in it across Close and Open: the Loop
// starts and stops processing it by its state changes, so reopening does not need Add again.
type Loop struct {
	ptr *C.tll_processor_loop_t
}

// Create new Loop
func NewLoop() *Loop {
	if ptr := C.tll_processor_loop_new_cfg(nil); ptr != nil {
		return &Loop{ptr}
	}
	return nil
}

// Create new Loop with parameters, config parameter is consumed and may be nil
func NewLoopCfg(cfg *ConstConfig) *Loop {
	if cfg == nil {
		return NewLoop()
	}
	defer cfg.Free()
	if ptr := C.tll_processor_loop_new_cfg(cfg.ptr); ptr != nil {
		return &Loop{ptr}
	}
	return nil
}

// Free destroys the Loop. Do not call it while another goroutine is inside Step. Channels that
// are still in the Loop are detached from it; they are not closed or released.
func (self *Loop) Free() {
	C.tll_processor_loop_free(self.ptr)
	self.ptr = nil
}

// Add puts the channel and its children into the Loop and returns 0 on success. A channel that
// is destroyed with Channel.Free is removed from the Loop automatically; Del is for taking a
// channel out of a Loop that stays alive and keeps the channel.
func (self Loop) Add(c Channel) int {
	return int(C.tll_processor_loop_add(self.ptr, c.ptr))
}

// Del removes the channel and its children from the Loop.
func (self Loop) Del(c Channel) int {
	return int(C.tll_processor_loop_del(self.ptr, c.ptr))
}

// Step waits up to timeout for activity and processes it once: channel callbacks are invoked
// from here, on the calling goroutine. Keep all calls for one Loop on one goroutine. The result
// is 0; libtll currently never reports an error from step, so a timeout is not an error.
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
