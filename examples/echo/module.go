package main

import "C"
import "github.com/shramov/tll-go/tll"

//export tll_channel_module
func tll_channel_module() uintptr {
	return module.Ptr()
}

var module = tll.NewModule(tll.CreateImpl[*Null](), tll.CreateImpl[*Echo]())

func main() {}
