package binder

// #cgo pkg-config: tll
/*
#include <tll/scheme/types.h>
*/
import "C"
import "bytes"
import "encoding/binary"
import "errors"
import "math"
import "time"
import "unsafe"

type Binder struct {
	data []byte
}

func NewBinder(ptr []byte) Binder { return Binder{ptr} }

func (self Binder) View(off uint) Binder { return Binder{self.data[off:]} }

func (self Binder) Int8(off uint) int8       { return int8(self.Uint8(off)) }
func (self Binder) Int16(off uint) int16     { return int16(self.Uint16(off)) }
func (self Binder) Int32(off uint) int32     { return int32(self.Uint32(off)) }
func (self Binder) Int64(off uint) int64     { return int64(self.Uint64(off)) }
func (self Binder) Uint8(off uint) uint8     { return self.data[off] }
func (self Binder) Uint16(off uint) uint16   { return binary.LittleEndian.Uint16(self.data[off:]) }
func (self Binder) Uint32(off uint) uint32   { return binary.LittleEndian.Uint32(self.data[off:]) }
func (self Binder) Uint64(off uint) uint64   { return binary.LittleEndian.Uint64(self.data[off:]) }
func (self Binder) Float64(off uint) float64 { return math.Float64frombits(self.Uint64(off)) }

func (self Binder) SetInt8(off uint, v int8)       { self.SetUint8(off, uint8(v)) }
func (self Binder) SetInt16(off uint, v int16)     { self.SetUint16(off, uint16(v)) }
func (self Binder) SetInt32(off uint, v int32)     { self.SetUint32(off, uint32(v)) }
func (self Binder) SetInt64(off uint, v int64)     { self.SetUint64(off, uint64(v)) }
func (self Binder) SetUint8(off uint, v uint8)     { self.data[off] = v }
func (self Binder) SetUint16(off uint, v uint16)   { binary.LittleEndian.PutUint16(self.data[off:], v) }
func (self Binder) SetUint32(off uint, v uint32)   { binary.LittleEndian.PutUint32(self.data[off:], v) }
func (self Binder) SetUint64(off uint, v uint64)   { binary.LittleEndian.PutUint64(self.data[off:], v) }
func (self Binder) SetFloat64(off uint, v float64) { self.SetUint64(off, math.Float64bits(v)) }

func (self Binder) ByteString(off uint, size uint) string {
	slice := self.data[off : off+size]
	if idx := bytes.IndexByte(slice, 0); idx != -1 {
		slice = slice[:idx]
	}
	return string(slice)
}

func (self Binder) SetByteString(data string, off uint, maxsize uint) error {
	size := uint(len(data))
	if size > maxsize {
		return errors.New("String too large")
	}
	slice := self.data[off : off+maxsize]
	copy(slice, data)
	for i := size; i < maxsize; i++ {
		slice[i] = 0
	}
	return nil
}

func (self Binder) PointerDefault(off uint) *PointerDefault {
	return (*PointerDefault)(unsafe.Pointer(&self.data[off]))
}
func (self Binder) PointerLegacyShort(off uint) *PointerLegacyShort {
	return (*PointerLegacyShort)(unsafe.Pointer(&self.data[off]))
}
func (self Binder) PointerLegacyLong(off uint) *PointerLegacyLong {
	return (*PointerLegacyLong)(unsafe.Pointer(&self.data[off]))
}

func (self Binder) stringPtr(off uint, ptr PointerImpl) string {
	size := ptr.Size()
	if size == 0 {
		return ""
	}
	offset := ptr.Offset()
	return string(self.data[off+offset : off+offset+size-1])
}

func (self Binder) String(off uint) string {
	return self.stringPtr(off, self.PointerDefault(off))
}

func (self Binder) StringLS(off uint) string {
	return self.stringPtr(off, self.PointerLegacyShort(off))
}

func (self Binder) StringLL(off uint) string {
	return self.stringPtr(off, self.PointerLegacyLong(off))
}

type PointerDefault C.tll_scheme_offset_ptr_t
type PointerLegacyShort C.tll_scheme_offset_ptr_legacy_short_t
type PointerLegacyLong C.tll_scheme_offset_ptr_legacy_long_t

type PointerImpl interface {
	Offset() uint
	Entity(size uint) uint
	Size() uint
}

func (self PointerDefault) Offset() uint {
	if self.entity != 0xff || self.offset == 0 {
		return uint(self.offset)
	}
	return uint(self.offset + 4)
}
func (self *PointerDefault) Entity(size uint) uint {
	if e := uint(self.entity); e != 0xff {
		return e
	}
	if self.offset == 0 {
		return size
	}
	return uint(*(*uint32)(unsafe.Add(unsafe.Pointer(&self.offset), self.offset)))
}
func (self *PointerDefault) Size() uint {
	slice := unsafe.Slice(&self.offset, 2)
	return uint(slice[1] & 0xffffff)
}

func (self PointerLegacyShort) Offset() uint          { return uint(self.offset) }
func (self PointerLegacyShort) Entity(size uint) uint { return size }
func (self PointerLegacyShort) Size() uint            { return uint(self.size) }

func (self PointerLegacyLong) Offset() uint          { return uint(self.offset) }
func (self PointerLegacyLong) Entity(size uint) uint { return uint(self.entity) }
func (self PointerLegacyLong) Size() uint            { return uint(self.size) }

func DurationFrom(v, mul, div int64) time.Duration {
	return time.Duration(v * (mul * 1000000000 / div))
}

func DurationInto(v time.Duration, mul, div int64) int64 {
	return int64(v) / (mul * 1000000000 / div)
}

func TimeFrom(v, mul, div int64) time.Time {
	ns := v * (mul * 1000000000 / div)
	return time.Unix(ns/1000000000, ns%1000000000).In(time.UTC)
}

func TimeInto(v time.Time, mul, div int64) int64 {
	return v.UnixNano() / (mul * 1000000000 / div)
}
