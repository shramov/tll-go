package tll

// #cgo pkg-config: tll
/*
#include <tll/scheme.h>
*/
import "C"

type Scheme struct{ ptr *C.tll_scheme_t }
type SchemeMessage struct{ ptr *C.tll_scheme_message_t }

func NewScheme(url string) *Scheme {
	ptr := C.tll_scheme_load(C._GoStringPtr(url), C.int(len(url)))
	if ptr == nil {
		return nil
	}
	return &Scheme{ptr}
}

func (self *Scheme) Free() {
	C.tll_scheme_unref(self.ptr)
	self.ptr = nil
}

func (self *Scheme) Ref() *Scheme {
	if self.ptr == nil {
		return nil
	}
	return &Scheme{C.tll_scheme_ref(self.ptr)}
}

func (self *Scheme) Copy() *Scheme {
	if self.ptr == nil {
		return nil
	}
	return &Scheme{C.tll_scheme_copy(self.ptr)}
}

func (self *Scheme) Get(name string) *SchemeMessage {
	m := SchemeMessage{nil}
	for ptr := self.ptr.messages; ptr != nil; ptr = ptr.next {
		m.ptr = ptr
		if m.Name() == name {
			return &m
		}
	}
	return nil
}

func (self *Scheme) GetById(id int) *SchemeMessage {
	for ptr := self.ptr.messages; ptr != nil; ptr = ptr.next {
		if ptr.msgid != 0 && ptr.msgid == C.int(id) {
			return &SchemeMessage{ptr}
		}
	}
	return nil
}

func (self *Scheme) Messages() map[string]SchemeMessage {
	r := make(map[string]SchemeMessage)
	for ptr := self.ptr.messages; ptr != nil; ptr = ptr.next {
		m := SchemeMessage{ptr}
		r[m.Name()] = m
	}
	return r
}

func (self *SchemeMessage) MsgId() int {
	return int(self.ptr.msgid)
}

func (self *SchemeMessage) Name() string {
	return C.GoString(self.ptr.name)
}

func (self *SchemeMessage) Size() uint {
	return uint(self.ptr.size)
}
