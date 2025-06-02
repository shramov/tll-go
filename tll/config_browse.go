package tll

// #cgo pkg-config: tll
/*
#include <tll/config.h>

_GoString_ GoStringN(char *, int);
extern int goBrowseCallback(char *, int, tll_config_t *, void * data);
*/
import "C"
import "runtime"
import "unsafe"

type BrowsePair struct {
	Key string
	Cfg ConstConfig
}

type BrowseResult struct {
	List []BrowsePair
}

func (self *BrowseResult) Unref() {
	for _, i := range self.List {
		i.Cfg.Unref()
	}
}

//export goBrowseCallback
func goBrowseCallback(key *C.char, klen C.int, cfg *C.tll_config_t, data *C.void) C.int {
	r := (*BrowseResult)(unsafe.Pointer(data))
	r.List = append(r.List, BrowsePair{C.GoStringN(key, klen), ConstConfig{cfg}.Ref()})
	return 0
}

func (self ConstConfig) Browse(mask string) BrowseResult {
	r := BrowseResult{} //make([]BrowseResult, 0)}
	pinner := runtime.Pinner{}
	pinner.Pin(&r)
	defer pinner.Unpin()
	C.tll_config_browse(self.ptr, C._GoStringPtr(mask), C.int(len(mask)), C.tll_config_callback_t(C.goBrowseCallback), unsafe.Pointer(&r))
	return r
}
