package tll

import "context"
import "testing"
import "time"

func TestLoop(t *testing.T) {
	ctx := Context{}
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	c := ctx.Channel("zero://;name=test;dump=frame")
	defer c.Free()

	loop.Add(*c)
	count := 0
	c.CallbackAdd(func(c Channel, m Message) int {
		println("Tick")
		count++
		return 0
	}, MessageMaskData)
	c.Open()
	loop.Step(time.Duration(0))
	assertEqual(t, count, 1)
	loop.Step(time.Duration(0))
	assertEqual(t, count, 2)
}

func TestLoopChan(t *testing.T) {
	ctx := Context{}
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	c := ctx.Channel("zero://;name=test;dump=frame")
	defer c.Free()

	loop.Add(*c)
	c.Open()
	context, cancel := context.WithCancel(context.Background())
	ch := c.ChanAdd(context, MessageMaskData)
	done := make(chan int, 1)
	go func() {
		count := 0
		defer func() {
			cancel()
			loop.SetStop(1)
		}()
		for m := range ch {
			println("Tick", len(m.Data))
			count++
			if count == 10 {
				break
			}
		}
		done <- count
	}()
	loop.Run(time.Millisecond)
	r := <-done
	println(r)
}
