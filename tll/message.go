package tll

// #cgo pkg-config: tll
/*
#include <tll/channel.h>
*/
import "C"
import "unsafe"
import "runtime"

type MessageType int

const (
	MessageData    int = C.TLL_MESSAGE_DATA
	MessageState   int = C.TLL_MESSAGE_STATE
	MessageControl int = C.TLL_MESSAGE_CONTROL
)

const (
	MessageMaskData    uint = C.TLL_MESSAGE_MASK_DATA
	MessageMaskState   uint = C.TLL_MESSAGE_MASK_STATE
	MessageMaskControl uint = C.TLL_MESSAGE_MASK_CONTROL
	MessageMaskChannel uint = C.TLL_MESSAGE_MASK_CHANNEL
	MessageMaskAll     uint = C.TLL_MESSAGE_MASK_ALL
)

type Message struct {
	ptr *C.tll_msg_t
}

func (self Message) Type() int {
	return int(self.ptr._type)
}

func (self Message) MsgId() int {
	return int(self.ptr.msgid)
}

func (self Message) Seq() int64 {
	return int64(self.ptr.seq)
}

func (self Message) Data() []byte {
	return unsafe.Slice((*byte)(self.ptr.data), self.ptr.size)
}

func (self Message) Addr() int64 {
	return *(*int64)(unsafe.Pointer(&self.ptr.addr[0]))
}

func (self Message) Copy() GoMessage {
	if self.ptr == nil {
		return GoMessage{}
	}
	r := GoMessage{Id: int(self.ptr.msgid), Type: int(self.ptr._type), Addr: self.Addr()}
	if self.ptr.data != nil {
		r.Data = C.GoBytes(self.ptr.data, C.int(self.ptr.size))
	}
	return r
}

type GoMessage struct {
	Type int
	Id   int
	Seq  int64
	Data []byte
	Addr int64
}

func (self *GoMessage) AsMsg(pinner *runtime.Pinner) Message {
	ptr := &C.tll_msg_t{
		msgid: C.int(self.Id),
		_type: C.short(self.Type),
		seq:   C.longlong(self.Seq),
		size:  C.size_t(len(self.Data)),
	}
	*(*int64)(unsafe.Pointer(&ptr.addr[0])) = self.Addr
	if len(self.Data) != 0 {
		ptr.data = unsafe.Pointer(&self.Data[0])
		pinner.Pin(ptr.data)
	}
	pinner.Pin(&ptr)
	return Message{ptr}
}
