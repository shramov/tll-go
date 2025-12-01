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
	cb := cgo.Handle(data).Value().(*CallbackHandle)
	return C.int(cb.cb(Channel{c}, Message{m}))
}

//export GoStateCallback
func GoStateCallback(c *C.tll_channel_t, m *C.tll_msg_t, data C.uintptr_t) C.int {
	if m.msgid != C.int(StateDestroy) {
		return 0
	}
	h := cgo.Handle(data).Value().(*CallbackHandle)
	h.Free()
	return 0
}

type Callback func(Channel, Message) int

type CallbackHandle struct {
	cb      Callback
	channel *Channel
	handle  cgo.Handle
}

func (self *CallbackHandle) Free() {
	if self.channel == nil {
		return
	}
	C.tll_channel_callback_del(self.channel.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(self.handle), C.unsigned(MessageMaskAll))
	C.tll_channel_callback_del(self.channel.ptr, C.tll_channel_callback_t(C.GoStateCallback), unsafe.Pointer(self.handle), C.unsigned(MessageMaskState))
	self.handle.Delete()
	self.channel = nil
	self.cb = nil
}

func (self Channel) CallbackAdd(cb Callback, mask uint) *CallbackHandle {
	cbh := CallbackHandle{cb, &self, 0}
	h := cgo.NewHandle(&cbh)
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(mask)) != 0 {
		h.Delete()
		return nil
	}
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoStateCallback), unsafe.Pointer(h), C.unsigned(MessageMaskState)) != 0 {
		C.tll_channel_callback_del(self.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(h), C.unsigned(mask))
		h.Delete()
		return nil
	}
	cbh.handle = h
	return &cbh
}
