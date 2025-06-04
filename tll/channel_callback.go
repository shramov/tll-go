package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
extern int GoCallback(tll_channel_t *, tll_msg_t *, uintptr_t);
*/
import "C"
import "runtime/cgo"
import "unsafe"

//export GoCallback
func GoCallback(c *C.tll_channel_t, m *C.tll_msg_t, data C.uintptr_t) C.int {
	cb := cgo.Handle(data).Value().(Callback)
	return C.int(cb(Channel{c}, Message{m}))
}

type Callback func(Channel, Message) int

func (self Channel) CallbackAdd(cb Callback, mask uint32) int {
	h := cgo.NewHandle(cb)
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(mask)) != 0 {
		return -1
	}
	return 0
}
