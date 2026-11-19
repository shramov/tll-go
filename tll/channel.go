package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
*/
import "C"

import "runtime"
import "syscall"

// Context is a reference counted libtll channel context: it creates channels and keeps their
// registry, configuration defaults and loaded schemes.
//
// Every Channel created from a Context holds its own reference to it. Context.Free drops only
// the reference of the caller, so the Context is destroyed after the last channel is released
// with Channel.Free, not before.
type Context struct {
	ptr *C.tll_channel_context_t
}

// Channel is a libtll channel.
//
// A Channel is created by Context.Channel or Context.ChannelCfg and lives until Channel.Free;
// closing it does not release it. Methods take Channel by value, so copies of a Channel share
// the same C pointer and all of them dangle after Free. A channel is meant to be created once
// and opened and closed repeatedly; see the package documentation.
type Channel struct {
	ptr *C.tll_channel_t
	pinner runtime.Pinner
}

// NewContext creates a new context with default settings. Release it with Context.Free.
func NewContext() Context {
	return Context{C.tll_channel_context_new(nil)}
}

// NewContextCfg creates a new context with cfg as its defaults (for example
// resolve.request or resolve.mode). The context takes its own reference to cfg, so the caller
// still owns cfg and releases it with Config.Free.
func NewContextCfg(cfg Configlike) Context {
	return Context{C.tll_channel_context_new(cfg.AsConstConfig().ptr)}
}

func (ctx Context) Ref() Context { return Context{C.tll_channel_context_ref(ctx.ptr)} }

// Free drops the reference held by this Context value. The context itself is destroyed only
// when no channel created from it is left, so release the channels first with Channel.Free.
func (ctx *Context) Free() {
	C.tll_channel_context_free(ctx.ptr)
	ctx.ptr = nil
}

// Channel creates a channel from url. It returns nil on error. The channel is created in
// StateClosed; it has to be opened with Open or OpenCfg and, when it is no longer needed,
// released with Channel.Free.
func (ctx Context) Channel(url string) *Channel {
	ptr := C.tll_channel_new(ctx.ptr, C._GoStringPtr(url), C.size_t(len(url)), nil, nil)
	if ptr == nil {
		return nil
	}
	return &Channel{ptr, runtime.Pinner{}}
}

func (ctx Context) ChannelCfg(cfg Configlike) *Channel {
	ptr := C.tll_channel_new_url(ctx.ptr, cfg.AsConstConfig().ptr, nil, nil)
	if ptr == nil {
		return nil
	}
	return &Channel{ptr, runtime.Pinner{}}
}

func (self Context) Get(name string) *Channel {
	ptr := C.tll_channel_get(self.ptr, C._GoStringPtr(name), C.int(len(name)))
	if ptr == nil {
		return nil
	}
	return &Channel{ptr, runtime.Pinner{}}
}

// Free destroys the channel: it force-closes it if it is not closed yet, removes it from the
// context, releases its resources and drops the reference it holds to the Context. Without it
// a closed channel, and the Context behind it, stay in memory. The channel must not be used
// afterwards.
//
// A Loop forgets a destroyed channel by itself, so Loop.Del is not needed first when Free is
// called on the goroutine that steps the Loop. If another goroutine is inside Loop.Step, stop
// it before calling Free.
func (self *Channel) Free() {
	C.tll_channel_free(self.ptr)
	self.ptr = nil
}

// Open opens the channel without parameters. The channel must be in StateClosed: after an
// error (StateError) close it first, after a non-forced Close wait until it reaches StateClosed
// (step the Loop) or use CloseForce. A successful return does not mean StateActive: channels
// such as tcp reach it only while the Loop is stepped.
func (self Channel) Open() error {
	return cint2error(C.tll_channel_open(self.ptr, nil, 0))
}

// OpenCfg opens the channel with open parameters (for example mode=block;block=0 for a stream
// client). cfg is only read and stays owned by the caller; nil means no parameters.
func (self Channel) OpenCfg(cfg *ConstConfig) error {
	if cfg == nil {
		return cint2error(C.tll_channel_open_cfg(self.ptr, nil))
	}
	return cint2error(C.tll_channel_open_cfg(self.ptr, cfg.ptr))
}

// Close starts closing the channel. It does not release the channel (see Free). Closing may be
// asynchronous: the channel can stay in StateClosing until the Loop is stepped, and a second
// non-forced Close in that state is ignored. Use CloseForce(true) when the channel has to be
// closed immediately, for example before reopening it after an error.
func (self Channel) Close() error {
	return cint2error(C.tll_channel_close(self.ptr, 0))
}

// CloseForce closes the channel; with force it closes immediately: the channel is in
// StateClosed when the call returns, it cannot fail and it is valid in any state, including
// StateError and StateClosing. This is what Free does before destroying a channel.
func (self Channel) CloseForce(force bool) error {
	fi := 0
	if force {
		fi = 1
	}
	return cint2error(C.tll_channel_close(self.ptr, C.int(fi)))
}

func (self Channel) Name() string {
	return C.GoString(C.tll_channel_name(self.ptr))
}

// State returns the lifecycle state: StateClosed, StateOpening, StateActive, StateClosing or
// StateError. Only Close can be called on a channel in StateError.
func (self Channel) State() State {
	return State(self.ptr.internal.state)
}

func (self Channel) SchemeData() *Scheme    { return self.Scheme(MessageData) }
func (self Channel) SchemeControl() *Scheme { return self.Scheme(MessageControl) }

func (self Channel) Scheme(type_ int) *Scheme {
	if ptr := C.tll_channel_scheme(self.ptr, C.int(type_)); ptr != nil {
		return &Scheme{ptr}
	} else {
		return nil
	}
}

func (self Channel) Post(m Message) error {
	return cint2error(C.tll_channel_post(self.ptr, m.ptr, 0))
}

func (self Channel) PostGo(m GoMessage) error {
	cmsg := m.AsMsg(&self.pinner)
	defer self.pinner.Unpin()
	return self.Post(cmsg)
}

func (self Channel) Process() error {
	if r := int(C.tll_channel_process(self.ptr, 0, 0)); r == 0 || r == int(syscall.EAGAIN) {
		return nil
	} else {
		return Errno(r)
	}
}
