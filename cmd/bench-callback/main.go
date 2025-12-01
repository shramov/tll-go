package main

import "fmt"
import "testing"

import "github.com/shramov/tll-go/tll"

func BenchmarkCallback(b *testing.B) {
	ctx := tll.NewContext()
	defer ctx.Free()
	c := ctx.Channel("zero://;name=test")
	defer c.Free()

	count := 0
	c.CallbackAdd(func(c tll.Channel, m tll.Message) int {
		count += 1
		return 0
	}, tll.MessageMaskData)
	c.Open()

	for b.Loop() {
		c.Process()
	}
}

func main() {
	tll.LoggerConfigMap(map[string]string{"type": "spdlog", "levels.tll": "warning"})
	fmt.Println("Callback: ", testing.Benchmark(BenchmarkCallback))
}
