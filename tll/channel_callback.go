package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
extern int GoCallback(tll_channel_t *, tll_msg_t *, uintptr_t);
extern int GoStateCallback(tll_channel_t *, tll_msg_t *, uintptr_t);
*/
import "C"
import "runtime/cgo"
import "unsafe"

//export GoCallback
func GoCallback(c *C.tll_channel_t, m *C.tll_msg_t, data C.uintptr_t) C.int {
	cb := cgo.Handle(data).Value().(Callback)
	return C.int(cb(Channel{c}, Message{m}))
}

//export GoStateCallback
func GoStateCallback(c *C.tll_channel_t, m *C.tll_msg_t, data C.uintptr_t) C.int {
	if m.msgid != C.int(StateDestroy) { return 0 }
	h := cgo.Handle(data)
	C.tll_channel_callback_del(c, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(MessageMaskAll));
	C.tll_channel_callback_del(c, C.tll_channel_callback_t(C.GoStateCallback), unsafe.Pointer(h), C.unsigned(MessageMaskState));
	return 0
}

type Callback func(Channel, Message) int

func (self Channel) CallbackAdd(cb Callback, mask uint) int {
	h := cgo.NewHandle(cb)
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(mask)) != 0 {
		h.Delete()
		return -1
	}
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoStateCallback), unsafe.Pointer(h), C.unsigned(MessageMaskState)) != 0 {
		C.tll_channel_callback_del(self.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(mask));
		h.Delete()
		return -1
	}
	return 0
}
