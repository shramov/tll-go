package tll

import "testing"

func TestCreate(t *testing.T) {
	ctx := Context{}
	c := ctx.Channel("zero://;name=test;dump=frame")
	println(c.Name())
	c.CallbackAdd(func(c Channel, m Message) int {
		println("Tick")
		return 0
	}, MessageMaskAll)
	c.Open()
	c.Process()
	c.Process()
	c.Process()
	c.Free()
}
