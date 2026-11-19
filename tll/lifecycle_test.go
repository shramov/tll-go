package tll

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// closedPort returns a local TCP port that nothing listens on, so that a client connecting to
// it ends in StateError.
func closedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

func tcpClient(port int, name string) string {
	return fmt.Sprintf("tcp://127.0.0.1:%d;mode=client;name=%s;dump=no", port, name)
}

// stepUntil steps the loop until cond holds or about a second passes.
func stepUntil(loop *Loop, cond func() bool) bool {
	for i := 0; i < 200; i++ {
		if cond() {
			return true
		}
		loop.Step(5 * time.Millisecond)
	}
	return cond()
}

func TestChannelReopensAfterConnectError(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	ch := ctx.Channel(tcpClient(closedPort(t), "client"))
	if ch == nil {
		t.Fatal("channel is not created")
	}
	defer ch.Free()
	assertEqual(t, loop.Add(*ch), 0)

	for attempt := 0; attempt < 3; attempt++ {
		assertEqual(t, ch.State(), StateClosed)
		if err := ch.Open(); err != nil {
			t.Fatalf("attempt %d: open: %v", attempt, err)
		}
		if !stepUntil(loop, func() bool { return ch.State() == StateError }) {
			t.Fatalf("attempt %d: state %v, want Error", attempt, ch.State())
		}
		// A channel in Error accepts nothing but Close.
		if err := ch.Open(); err == nil {
			t.Fatalf("attempt %d: open in Error succeeded", attempt)
		}
		_ = ch.CloseForce(true)
		assertEqual(t, ch.State(), StateClosed) // forced close is immediate
	}
}

func TestCloseForceIsValidInAnyState(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	ch := ctx.Channel("zero://;name=zero")
	defer ch.Free()

	if err := ch.CloseForce(true); err != nil { // already closed: silently ignored
		t.Fatalf("close of a closed channel: %v", err)
	}
	assertEqual(t, ch.State(), StateClosed)
	if err := ch.Open(); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, ch.State(), StateActive)
	if err := ch.CloseForce(true); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, ch.State(), StateClosed)
}

func TestLoopFollowsChannelAcrossReopen(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	ch := ctx.Channel("zero://;name=zero")
	defer ch.Free()
	assertEqual(t, loop.Add(*ch), 0) // added once, never again
	count := 0
	h := ch.CallbackAdd(func(Channel, Message) int { count++; return 0 }, MessageMaskData)
	defer h.Free()

	for cycle := 1; cycle <= 3; cycle++ {
		if err := ch.Open(); err != nil {
			t.Fatal(err)
		}
		before := count
		loop.Step(0)
		if count <= before {
			t.Fatalf("cycle %d: no messages after reopen", cycle)
		}
		_ = ch.CloseForce(true)
		before = count
		loop.Step(0)
		assertEqual(t, count, before) // a closed channel is not processed
	}
}

func TestFreeRemovesChannelFromContextAndLoop(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	ch := ctx.Channel("zero://;name=gone")
	assertEqual(t, loop.Add(*ch), 0)
	if err := ch.Open(); err != nil {
		t.Fatal(err)
	}
	if ctx.Get("gone") == nil {
		t.Fatal("channel is not registered in the context")
	}

	ch.Free() // no Loop.Del: the loop forgets a destroyed channel by itself
	if ctx.Get("gone") != nil {
		t.Fatal("freed channel is still registered in the context")
	}
	assertEqual(t, loop.Step(0), 0) // must not touch the freed channel
}

func TestLoopStepReturnsZeroOnTimeout(t *testing.T) {
	loop := NewLoop()
	defer loop.Free()
	assertEqual(t, loop.Step(time.Millisecond), 0)
}

func TestCallbackFreeStopsDelivery(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	ch := ctx.Channel("zero://;name=zero")
	defer ch.Free()
	loop.Add(*ch)
	count := 0
	h := ch.CallbackAdd(func(Channel, Message) int { count++; return 0 }, MessageMaskData)
	if h == nil {
		t.Fatal("callback is not added")
	}
	if err := ch.Open(); err != nil {
		t.Fatal(err)
	}
	loop.Step(0)
	if count == 0 {
		t.Fatal("no messages delivered")
	}
	h.Free()
	before := count
	loop.Step(0)
	loop.Step(0)
	assertEqual(t, count, before)
}

func TestCallbackFreeInsideCallback(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	loop := NewLoop()
	defer loop.Free()
	ch := ctx.Channel("zero://;name=zero")
	defer ch.Free()
	loop.Add(*ch)
	count := 0
	var h *CallbackHandle
	h = ch.CallbackAdd(func(Channel, Message) int {
		count++
		h.Free() // libtll itself removes callbacks from callbacks
		return 0
	}, MessageMaskData)
	if err := ch.Open(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		loop.Step(0)
	}
	assertEqual(t, count, 1)
}

// directPair returns a master and a slave direct channel: whatever is posted to one is passed
// to the callbacks of the other synchronously, inside Post.
func directPair(t *testing.T, ctx Context) (master, slave *Channel) {
	t.Helper()
	master = ctx.Channel("direct://;name=master")
	slave = ctx.Channel("direct://;name=slave;master=master")
	if master == nil || slave == nil {
		t.Fatal("direct channels are not created")
	}
	for _, c := range []*Channel{master, slave} {
		if err := c.Open(); err != nil {
			t.Fatal(err)
		}
	}
	return master, slave
}

func TestMessageCopyOutlivesCallback(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	master, slave := directPair(t, ctx)
	defer master.Free()
	defer slave.Free()

	var got []GoMessage
	h := master.CallbackAdd(func(_ Channel, m Message) int {
		got = append(got, m.Copy()) // the message itself is valid only inside the callback
		return 0
	}, MessageMaskData)
	defer h.Free()

	payload := []byte("payload")
	if err := slave.PostGo(GoMessage{Type: MessageData, Id: 7, Seq: 5, Data: payload}); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X' // the copy must not alias the caller's buffer either
	assertEqual(t, len(got), 1)
	assertEqual(t, got[0].Id, 7)
	assertEqual(t, got[0].Seq, int64(5))
	assertEqual(t, string(got[0].Data), "payload")
}

func TestChanAddBufferedDeliversCopies(t *testing.T) {
	ctx := NewContext()
	defer ctx.Free()
	master, slave := directPair(t, ctx)
	defer master.Free()
	defer slave.Free()

	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	frames := master.ChanAddBuffered(goCtx, 4, MessageMaskData)
	for i := 0; i < 3; i++ {
		if err := slave.PostGo(GoMessage{Type: MessageData, Id: i, Data: []byte(fmt.Sprintf("m%d", i))}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 3; i++ {
		select {
		case m := <-frames:
			assertEqual(t, m.Id, i)
			assertEqual(t, string(m.Data), fmt.Sprintf("m%d", i))
		case <-time.After(time.Second):
			t.Fatalf("message %d is not delivered", i)
		}
	}
}

func TestContextCfgDoesNotConsumeConfig(t *testing.T) {
	cfg := NewConfig()
	cfg.Set("resolve.mode", "always")
	ctx := NewContextCfg(cfg)

	cfg.Set("later", "1") // the caller still owns cfg and can keep using it
	assertEqual(t, *cfg.Get("later"), "1")
	cfg.Free() // and has to release it; the context holds its own reference

	ch := ctx.Channel("null://;name=null")
	if ch == nil {
		t.Fatal("channel is not created in a context made from a freed config")
	}
	ch.Free()
	ctx.Free()
}

func residentMB() (int, error) {
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

// Recreating a Context, Loop and channel for every attempt and releasing them must not grow
// the process. A channel that is only closed keeps its Context alive: about 0.5 MB per attempt
// with libtll 0.6.4, so 300 attempts would add roughly 150 MB against the 30 MB allowed here.
func TestRecreatedChannelIsReleased(t *testing.T) {
	if testing.Short() {
		t.Skip("memory check")
	}
	if _, err := residentMB(); err != nil {
		t.Skip("needs /proc/self/status: ", err)
	}
	port := closedPort(t)
	attempt := func() {
		ctx := NewContext()
		loop := NewLoop()
		ch := ctx.Channel(tcpClient(port, "client"))
		loop.Add(*ch)
		if err := ch.Open(); err == nil {
			stepUntil(loop, func() bool { return ch.State() == StateError })
		}
		_ = ch.Close()
		ch.Free()
		loop.Free()
		ctx.Free()
	}
	for i := 0; i < 30; i++ { // warm up allocator and caches
		attempt()
	}
	before, _ := residentMB()
	for i := 0; i < 300; i++ {
		attempt()
	}
	after, _ := residentMB()
	if after-before > 30 {
		t.Fatalf("resident memory grew by %d MB over 300 attempts", after-before)
	}
}
