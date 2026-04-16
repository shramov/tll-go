package main

import "context"
import "fmt"
import "testing"

import "github.com/shramov/tll-go/tll"

type B struct {
	*testing.B
}

func BenchmarkCallback(b *testing.B) {
	ctx := tll.NewContext()
	defer ctx.Free()
	c := ctx.Channel("zero://;name=test")
	defer c.Free()

	count := 0
	cbh := c.CallbackAdd(func(c tll.Channel, m tll.Message) int {
		count += 1
		return 0
	}, tll.MessageMaskData)
	defer cbh.Free()

	c.Open()

	wrapb := B{b}
	for wrapb.Loop() {
		c.Process()
	}
	if count != b.N {
		fmt.Printf("Count mismatch: iteration %d != callback calls %d\n", b.N, count)
	}
}

func BenchmarkChan(b *testing.B, bufsize int) {
	ctx := tll.NewContext()
	defer ctx.Free()
	c := ctx.Channel("zero://;name=test")
	defer c.Free()

	c.Open()

	count := 0
	cctx, cancel := context.WithCancel(context.Background())
	ch := c.ChanAddBuffered(cctx, bufsize, tll.MessageMaskData)
	defer cancel()


	done := make(chan bool)
	stop := false
	go func() {
		for !stop {
			c.Process()
		}
		done <- true
	}()

	wrapb := B{b}
	for wrapb.Loop() {
		//c.Process()
		<- ch
		count++
	}
	b.StopTimer()
	cancel()
	stop = true
	<-done

	if count != b.N {
		fmt.Printf("Count mismatch: iteration %d != callback calls %d\n", b.N, count)
	}
}

func main() {
	tll.LoggerConfigMap(map[string]string{"type": "spdlog", "levels.tll": "warning"})
	fmt.Println("Callback: ", testing.Benchmark(BenchmarkCallback))
	for _, s := range([]int{0, 1, 2, 4, 8, 16}) {
		fmt.Printf("Channel: %2d %v\n", s, testing.Benchmark(func(b *testing.B) { BenchmarkChan(b, s) }))
	}
}
