package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
*/
import "C"

type Message struct {
	ptr *C.tll_msg_t
}

type Context struct {
	ptr *C.tll_channel_context_t
}

type Channel struct {
	ptr *C.tll_channel_t
}

func NewContext() Context {
	return Context{C.tll_channel_context_new(nil)}
}

func (ctx *Context) Channel(url string) *Channel {
	ptr := C.tll_channel_new(ctx.ptr, C._GoStringPtr(url), C.size_t(len(url)), nil, nil)
	if ptr == nil {
		return nil
	}
	return &Channel{ptr}
}

func (ctx *Context) ChannelCfg(cfg *ConstConfig) *Channel {
	ptr := C.tll_channel_new_url(ctx.ptr, cfg.ptr, nil, nil)
	if ptr == nil {
		return nil
	}
	return &Channel{ptr}
}

func (self *Channel) Free() {
	C.tll_channel_free(self.ptr)
	self.ptr = nil
}

func (self *Channel) Open() int {
	return int(C.tll_channel_open(self.ptr, nil, 0))
}

func (self *Channel) OpenCfg(cfg *ConstConfig) int {
	return int(C.tll_channel_open_cfg(self.ptr, cfg.ptr))
}

func (self *Channel) Close() int {
	return int(C.tll_channel_close(self.ptr, 0))
}

func (self *Channel) CloseForce(force bool) int {
	fi := 0
	if force {
		fi = 1
	}
	return int(C.tll_channel_close(self.ptr, C.int(fi)))
}

func (self *Channel) Name() string {
	return C.GoString(C.tll_channel_name(self.ptr))
}

func (self *Channel) Post(m *Message) int {
	return int(C.tll_channel_post(self.ptr, m.ptr, 0))
}

func (self *Channel) Process() int {
	return int(C.tll_channel_process(self.ptr, 0, 0))
}
