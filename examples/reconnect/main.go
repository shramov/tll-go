// Command reconnect shows how to reconnect a channel and measures what each way costs.
//
// It connects a tcp client to a port nobody listens on, so every attempt ends in StateError,
// and prints the resident memory of the process (Linux only) every 500 attempts:
//
//	go run ./examples/reconnect -mode reuse                 # create once, Open and CloseForce
//	go run ./examples/reconnect -mode recreate -free=true   # new Context, Loop and Channel per attempt, released
//	go run ./examples/reconnect -mode recreate -free=false  # the same, but only Close: leaks a Context per attempt
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/shramov/tll-go/tll"
)

const url = "tcp://127.0.0.1:5999;mode=client;name=reconnect;dump=no"

// rssMB returns the resident memory of the process in MB; it reads /proc, so it needs Linux.
func rssMB() (int, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		var kb int
		if _, err := fmt.Sscanf(line, "VmRSS: %d kB", &kb); err == nil {
			return kb / 1024, nil
		}
	}
	return 0, fmt.Errorf("no VmRSS in /proc/self/status")
}

// attempt opens the channel once and steps the loop until it fails or becomes active.
func attempt(ch *tll.Channel, loop *tll.Loop) {
	if err := ch.Open(); err != nil {
		log.Printf("open: %v", err)
		return
	}
	for i := 0; i < 50; i++ {
		// libtll currently always returns 0 from step; keep the check for errors reported later.
		if rc := loop.Step(time.Millisecond); rc != 0 {
			log.Printf("step: %d", rc)
			return
		}
		if st := ch.State(); st == tll.StateError || st == tll.StateActive {
			return
		}
	}
}

func main() {
	mode := flag.String("mode", "reuse", "reuse or recreate")
	free := flag.Bool("free", true, "recreate mode: release the channel with Free (false leaks)")
	n := flag.Int("n", 3000, "number of attempts")
	flag.Parse()

	start, err := rssMB()
	if err != nil {
		log.Fatalf("this example measures memory through /proc/self/status and needs Linux: %v", err)
	}
	report := func(i int) {
		if i%500 == 0 {
			now, _ := rssMB() // one reading per line, so the two numbers agree
			fmt.Printf("%-9s attempt=%d rss=%dMB (+%d)\n", *mode, i, now, now-start)
		}
	}
	if *mode == "reuse" {
		ctx := tll.NewContext()
		defer ctx.Free()
		loop := tll.NewLoop()
		defer loop.Free()
		ch := ctx.Channel(url)
		if ch == nil {
			log.Fatal("cannot create channel")
		}
		// Runs before loop.Free and ctx.Free; the loop forgets a destroyed channel by itself.
		defer ch.Free()
		if rc := loop.Add(*ch); rc != 0 {
			log.Fatalf("loop.Add: %d", rc)
		}
		for i := 1; i <= *n; i++ {
			attempt(ch, loop)
			_ = ch.CloseForce(true) // allowed in any state, leaves the channel Closed
			report(i)
		}
		return
	}
	for i := 1; i <= *n; i++ {
		ctx := tll.NewContext()
		loop := tll.NewLoop()
		ch := ctx.Channel(url)
		if ch == nil {
			log.Fatal("cannot create channel")
		}
		loop.Add(*ch)
		attempt(ch, loop)
		_ = ch.Close()
		if *free {
			ch.Free()
		}
		loop.Free()
		ctx.Free()
		report(i)
	}
}
