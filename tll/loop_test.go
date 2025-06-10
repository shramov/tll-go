package tll

import "testing"
import "time"

func TestLoop(t *testing.T) {
	cfg := NewConfig()
	defer cfg.Unref()

	ctx := Context{}
	defer ctx.Free()
	loop := NewLoop(cfg.ConstConfig)
	defer loop.Free()
	c := ctx.Channel("zero://;name=test;dump=frame")
	defer c.Free()


	loop.Add(*c)
	count := 0
	c.CallbackAdd(func(c Channel, m Message) int {
		if m.Type() == MessageData {
			println("Tick")
			count++
		}
		return 0
	}, 0xFFFF)
	c.Open()
	loop.Step(time.Duration(0))
	assertEqual(t, count, 1)
	loop.Step(time.Duration(0))
	assertEqual(t, count, 2)
}
