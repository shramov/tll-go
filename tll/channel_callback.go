package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
extern int GoCallback(tll_channel_t *, tll_msg_t *, void *);
extern int GoStateCallback(tll_channel_t *, tll_msg_t *, void *);
*/
import "C"
import "context"
import "runtime"
import "runtime/cgo"
import "unsafe"

//export GoCallback
func GoCallback(c *C.tll_channel_t, m *C.tll_msg_t, data unsafe.Pointer) C.int {
	cb := (*cgo.Handle)(data).Value().(*CallbackHandle)
	return C.int(cb.cb(Channel{c, runtime.Pinner{}}, Message{m}))
}

//export GoStateCallback
func GoStateCallback(c *C.tll_channel_t, m *C.tll_msg_t, data unsafe.Pointer) C.int {
	if m.msgid != C.int(StateDestroy) {
		return 0
	}
	h := (*cgo.Handle)(data).Value().(*CallbackHandle)
	h.Free()
	return 0
}

type Callback func(Channel, Message) int

type CallbackHandle struct {
	cb      Callback
	channel *Channel
	pinner  runtime.Pinner
	handle  *cgo.Handle
}

// Free removes the callback from the channel. The handle is released automatically when the
// channel is destroyed, but a channel that is reused (opened and closed repeatedly) never is,
// so call Free when the callback is no longer needed.
//
// It is safe to call Free from inside a callback (libtll itself removes callbacks from callbacks,
// for example a Loop does it when a channel is destroyed). It is not safe to call it from
// another goroutine while the Loop that drives the channel is being stepped.
func (self *CallbackHandle) Free() {
	if self.channel == nil {
		return
	}
	C.tll_channel_callback_del(self.channel.ptr, C.tll_channel_callback_t(C.GoCallback), unsafe.Pointer(self.handle), C.unsigned(MessageMaskAll))
	C.tll_channel_callback_del(self.channel.ptr, C.tll_channel_callback_t(C.GoStateCallback), unsafe.Pointer(self.handle), C.unsigned(MessageMaskState))
	self.handle.Delete()
	self.pinner.Unpin()
	self.handle = nil
	self.channel = nil
	self.cb = nil
}

// CallbackAdd registers cb for the message types in mask (for example
// MessageMaskData|MessageMaskControl). The callback runs inside Loop.Step on the goroutine
// that steps the Loop, and the Message is valid only until cb returns: use Message.Copy to
// keep it. It returns nil on error. Release the returned handle with CallbackHandle.Free.
func (self Channel) CallbackAdd(cb Callback, mask uint) *CallbackHandle {
	cbh := CallbackHandle{cb, &self, runtime.Pinner{}, nil}
	handle := cgo.NewHandle(&cbh)
	cbh.handle = &handle
	cbh.pinner.Pin(cbh.handle)
	h := unsafe.Pointer(cbh.handle)
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoCallback), h, C.unsigned(mask)) != 0 {
		cbh.handle.Delete()
		cbh.pinner.Unpin()
		return nil
	}
	if C.tll_channel_callback_add(self.ptr, C.tll_channel_callback_t(C.GoStateCallback), h, C.unsigned(MessageMaskState)) != 0 {
		C.tll_channel_callback_del(self.ptr, C.tll_channel_callback_t(C.GoCallback), h, C.unsigned(mask))
		cbh.handle.Delete()
		cbh.pinner.Unpin()
		return nil
	}
	return &cbh
}

type ChanCallback struct {
	ch     chan GoMessage
	ctx    context.Context
	handle *CallbackHandle
}

func (self *ChanCallback) Push(c Channel, m Message) int {
	if self.ch == nil {
		return 0
	}
	select {
	case self.ch <- m.Copy():
		break
	case <-self.ctx.Done():
		self.handle.Free()
		close(self.ch)
		self.ch = nil
	}
	return 0
}

// ChanAddBuffered delivers copies of the messages to a Go channel with a buffer of size.
//
// Known limitations: the callback blocks the Loop (the whole Step) while the buffer is full and
// nobody reads, until ctx is done; the returned channel is closed only when a message arrives
// after ctx is done, so a reader can wait forever on a silent channel; and the handle is not
// returned, so the callback cannot be removed from a channel that is reused. For a long-lived
// channel use CallbackAdd with your own Go channel and release the handle.
func (self Channel) ChanAddBuffered(ctx context.Context, size int, mask uint) <-chan GoMessage {
	r := ChanCallback{ch: make(chan GoMessage, size), ctx: ctx}
	r.handle = self.CallbackAdd(r.Push, mask)
	return r.ch
}

func (self Channel) ChanAdd(ctx context.Context, mask uint) <-chan GoMessage { return self.ChanAddBuffered(ctx, 0, mask) }
