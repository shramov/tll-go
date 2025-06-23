package main

import "bytes"
import "flag"
import "fmt"
import "math/rand/v2"
import "runtime"
import "time"

import "github.com/shramov/tll-go/tll"

type Message struct {
	seq  int64
	data []byte
}

type Settings struct {
	io   string
	ctx  tll.Context
	base string
	data []Message
}

type Reader struct {
	Settings
	loop tll.Loop
	count uint64
}

func (self *Reader) onData(c tll.Channel, m tll.Message) int {
	msg := self.data[m.Seq()%int64(len(self.data))]
	if !bytes.Equal(msg.data, m.Data()) {
		println("Expected", msg.data, msg.data[0])
		println("Got", m.Data(), m.Data()[0])
		println("Error on seq", m.Seq())
		panic(fmt.Sprintf("Error on seq %d", m.Seq()))
	}
	self.count++
	return 0
}

func (self *Reader) Run() {
	cfg := tll.NewConfig()
	cfg.Set("poll", "no")
	self.loop = *tll.NewLoop(cfg.ConstConfig)
	defer self.loop.Free()

	r := self.ctx.Channel(fmt.Sprintf("file://%s/file.dat;autoclose=no;io=%s;dump=no;name=reader-%d", self.base, self.io, 0))
	self.loop.Add(*r)
	defer r.Free()
	r.CallbackAdd(self.onData, tll.MessageMaskData)
	r.CallbackAdd(func(c tll.Channel, m tll.Message) int {
		if tll.State(m.MsgId()) == tll.StateError {
			panic("Error in reader")
		}
		return 0
	}, tll.MessageMaskState)

	ocfg := tll.NewConfig()
	defer ocfg.Free()
	ocfg.Set("mode", "last")

	reopen := self.ctx.Channel(fmt.Sprintf("pub+mem://%s/reopen;dump=frame;name=reopen-%d", self.base, 0))
	self.loop.Add(*reopen)
	defer reopen.Free()
	reopen.CallbackAdd(func(c tll.Channel, m tll.Message) int {
		r.Close()
		if self.count > 0 {
			fmt.Printf("Checked %d messages\n", self.count)
		}
		self.count = 0
		r.OpenCfg(&ocfg.ConstConfig)
		return 0
	}, tll.MessageMaskData)
	reopen.Open()

	self.loop.Run(time.Millisecond)
}

func main() {
	s := Settings{}

	wio := flag.String("wio", "mmap", "writer io")
	wextra := flag.String("extra-space", "1mb", "writer extra space")
	flag.StringVar(&s.io, "rio", "mmap", "reader io")

	flag.Parse()

	lcfg := tll.NewConfig()
	lcfg.Set("type", "spdlog")
	lcfg.Set("levels.tll", "info")
	tll.LoggerConfig(lcfg.ConstConfig)

	s.ctx = tll.NewContext()
	defer s.ctx.Free()

	s.base = "/tmp/"
	s.data = make([]Message, 8192)
	seed := [32]byte{0, 1, 2, 3}
	prng := rand.NewChaCha8(seed)
	for i := range len(s.data) {
		m := Message{seq: int64(i)}
		m.data = make([]byte, rand.Int32N(1024))
		prng.Read(m.data)
		s.data[i] = m
	}

	wcfg := tll.LoadConfigData("url", "file:///tmp/file.dat;dir=w;name=writer;dump=no")
	wcfg.Set("io", *wio)
	wcfg.Set("extra-space", *wextra)
	w := s.ctx.ChannelCfg(wcfg.ConstConfig)
	reopen := s.ctx.Channel("pub+mem:///tmp/reopen;mode=server;name=reopen-signal")
	reopen.Open()

	ocfg := tll.NewConfig()
	defer ocfg.Free()
	ocfg.Set("overwrite", "yes")

	pinner := runtime.Pinner{}
	msg := tll.GoMessage{}
	empty := make([]byte, 0)
	reader := Reader{Settings: s}
	go reader.Run()
	for range 100 {
		w.OpenCfg(&ocfg.ConstConfig)

		msg.Data = empty
		cmsg := msg.AsMsg(&pinner)
		reopen.Post(cmsg)
		pinner.Unpin()
		for i := range 1000000 {
			m := s.data[i%8192]
			msg.Seq = int64(i)
			msg.Data = m.data
			cmsg := msg.AsMsg(&pinner)
			w.Post(cmsg)
			pinner.Unpin()
		}
		w.Close()
	}
	reader.loop.SetStop(1)
}
