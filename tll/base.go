package tll

// #cgo pkg-config: tll
/*
#include <tll/channel/impl.h>
#include <tll/channel/module.h>
#include <tll/logger.h>
#include <string.h>

static inline uintptr_t _impl_handle(tll_channel_impl_t *impl) { return (uintptr_t) impl->data; }

extern int _GoInit(tll_channel_t *, tll_config_t *, tll_channel_t *, tll_channel_context_t *);
extern void _GoFree(tll_channel_t *);
extern int _GoOpen(tll_channel_t *, tll_config_t *);
extern int _GoClose(tll_channel_t *, int);
extern int _GoProcess(tll_channel_t *, long, int);
extern int _GoPost(tll_channel_t *, tll_msg_t *, int);

static inline tll_channel_impl_t * _go_impl_alloc()
{
	tll_channel_impl_t * impl = malloc(sizeof(tll_channel_impl_t));
	memset(impl, 0, sizeof(*impl));
	impl->init = (int (*)(tll_channel_t *, const tll_config_t *, tll_channel_t *, tll_channel_context_t *))_GoInit;
	impl->free = _GoFree;
	impl->open = (int (*)(tll_channel_t *, const tll_config_t *))_GoOpen;
	impl->close = _GoClose;
	impl->process = _GoProcess;
	impl->post = (int (*)(tll_channel_t *, const tll_msg_t *, int))_GoPost;
	return impl;
}

static inline tll_channel_internal_t * _go_internal_alloc()
{
	tll_channel_internal_t * ptr = malloc(sizeof(tll_channel_internal_t));
	memset(ptr, 0, sizeof(*ptr));
	return ptr;
}

static inline tll_channel_impl_t ** alloc_impl(size_t size)
{
	tll_channel_impl_t ** r = malloc(sizeof(tll_channel_impl_t *) * size);
	memset(r, 0, sizeof(*r) * size);
	return r;
}

static inline void impl_set(tll_channel_impl_t ** ptr, size_t idx, tll_channel_impl_t *v)
{
	ptr[idx] = v;
}
*/
import "C"
import "runtime/cgo"
import "runtime"
import "fmt"
import "unsafe"
import "syscall"

type MessageLogFormat int

const (
	MessageLogDisable = C.TLL_MESSAGE_LOG_DISABLE
	MessageLogFrame   = C.TLL_MESSAGE_LOG_FRAME
	MessageLogText    = C.TLL_MESSAGE_LOG_TEXT
	MessageLogTextHex = C.TLL_MESSAGE_LOG_TEXT_HEX
	MessageLogScheme  = C.TLL_MESSAGE_LOG_SCHEME
	MessageLogAuto    = C.TLL_MESSAGE_LOG_AUTO
)

/*
type FromString interface {
	func FromString(string) (Parse, error)
}
*/

var (
	logFormatMap = map[string]MessageLogFormat{
		"no":       MessageLogDisable,
		"yes":      MessageLogAuto,
		"auto":     MessageLogAuto,
		"frame":    MessageLogFrame,
		"text":     MessageLogText,
		"text+hex": MessageLogTextHex,
		"scheme":   MessageLogScheme,
	}
)

func parseMessageLogFormat(str string) (MessageLogFormat, error) {
	if v, ok := logFormatMap[str]; ok {
		return v, nil
	}
	return MessageLogDisable, fmt.Errorf("Unknown dump value %s", str)
}

type Impl struct {
	impl *C.tll_channel_impl_t
}

type ChannelImpl interface {
	Protocol() string
	GetBase() *Base
	Init(cfg *ConstConfig, ctx *Context) ChannelImpl
	Free()
	Open(cfg *ConstConfig) int
	Close(bool) int
	Process() int
	Post(*Message) int
}

type Base struct {
	impl     ChannelImpl
	internal *C.tll_channel_internal_t
	pinner   runtime.Pinner
}

func (self *Base) InitInternal() {
	if self.internal != nil {
		return
	}
	self.pinner.Pin(self)
	self.internal = new(C.tll_channel_internal_t)
	self.pinner.Pin(self.internal)
	C.tll_channel_internal_init(self.internal)
}

func (self *Base) State() State {
	return State(self.internal.state)
}

func (self *Base) SetState(s State) {
	C.tll_channel_internal_set_state(self.internal, C.tll_state_t(s))
}

func (self *Base) Callback(m *Message) {
	C.tll_channel_callback(self.internal, m.ptr)
}

func (self *Base) CallbackData(m *Message) {
	C.tll_channel_callback_data(self.internal, m.ptr)
}

func (self *Base) ChildAdd(c *Channel, tag string) int {
	return int(C.tll_channel_internal_child_add(self.internal, c.ptr, nil, 0))
}

func (self *Base) ChildDel(c *Channel, tag string) int {
	return int(C.tll_channel_internal_child_del(self.internal, c.ptr, nil, 0))
}

func (self *Base) GetBase() *Base { return self }
func (*Base) Free()               {}

func (self *Base) Open(*ConstConfig) int {
	self.SetState(StateActive)
	return 0
}

func (self *Base) Close(bool) int {
	self.SetState(StateClosed)
	return 0
}

func (*Base) Process() int {
	return 0
}

func (self *Base) Post(m *Message) int {
	return 0
}

//export _GoInit
func _GoInit(c *C.tll_channel_t, ccfg *C.tll_config_t, master *C.tll_channel_t, context *C.tll_channel_context_t) C.int {
	h := cgo.Handle(C._impl_handle(c.impl)).Value().(ChannelImpl)
	cfg := ConstConfig{ccfg}
	data := h.Init(&cfg, &Context{context})
	if data == nil {
		return C.int(syscall.EINVAL)
	}

	base := data.GetBase()
	base.InitInternal()
	base.impl = data

	c.data = unsafe.Pointer(base)
	c.internal = base.internal
	c.internal.name = C.CString("tll.go")
	c.internal.logger = C.tll_logger_new(c.internal.name, -1)
	c.internal.self = c

	if s := cfg.Get("dump"); s != nil {
		if v, err := parseMessageLogFormat(*s); err == nil {
			c.internal.dump = C.tll_channel_log_msg_format_t(v)
		}
	}
	return 0
}

//export _GoFree
func _GoFree(c *C.tll_channel_t) {
	data := (*Base)(unsafe.Pointer(c.data))
	data.impl.Free()
	data.pinner.Unpin()
	C.tll_logger_free(c.internal.logger)
	C.free(unsafe.Pointer(c.internal.name))
}

//export _GoOpen
func _GoOpen(c *C.tll_channel_t, cfg *C.tll_config_t) C.int {
	data := (*Base)(unsafe.Pointer(c.data))
	data.SetState(StateOpening)
	return C.int(data.impl.Open(&ConstConfig{cfg}))
}

//export _GoClose
func _GoClose(c *C.tll_channel_t, force C.int) C.int {
	data := (*Base)(unsafe.Pointer(c.data))
	data.SetState(StateClosing)
	return C.int(data.impl.Close(force != 0))
}

//export _GoProcess
func _GoProcess(c *C.tll_channel_t, timeout C.long, flags C.int) C.int {
	data := (*Base)(unsafe.Pointer(c.data))
	return C.int(data.impl.Process())
}

//export _GoPost
func _GoPost(c *C.tll_channel_t, m *C.tll_msg_t, flags C.int) C.int {
	data := (*Base)(unsafe.Pointer(c.data))
	return C.int(data.impl.Post(&Message{m}))
}

func CreateImpl[I ChannelImpl]() *Impl {
	impl := new(Impl)
	impl.impl = C._go_impl_alloc()
	i := new(I)
	impl.impl.data = unsafe.Pointer(cgo.NewHandle(*i))
	impl.impl.name = C.CString((*i).Protocol())
	return impl
}

func (ctx *Context) register(impl *Impl) int {
	return int(C.tll_channel_impl_register(ctx.ptr, impl.impl, nil))
}

type CModule C.tll_channel_module_t

type Module struct {
	ptr    *CModule
	pinner runtime.Pinner
	impl   []*Impl
}

func (self *Module) Ptr() uintptr {
	return uintptr(unsafe.Pointer(self.ptr))
}

func NewModule(impls ...*Impl) Module {
	r := Module{}
	r.ptr = new(CModule)
	r.impl = impls
	r.ptr.impl = C.alloc_impl(C.size_t(len(impls) + 1))
	for i, impl := range r.impl {
		C.impl_set(r.ptr.impl, C.size_t(i), impl.impl)
	}
	return r
}
