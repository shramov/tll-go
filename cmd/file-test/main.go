package main

import "bytes"
import "flag"
import "fmt"
import "math/rand"
import "runtime"
import "time"

import "github.com/shramov/tll-go/tll"

type Message struct {
	seq   int64
	msgid int
	data  []byte
}

type Settings struct {
	io   string
	ctx  tll.Context
	base string
	data []Message
}

type Reader struct {
	Settings
	loop  tll.Loop
	count uint64
	index int
}

func (self *Reader) onData(c tll.Channel, m tll.Message) int {
	msg := self.data[m.Seq()%int64(len(self.data))]
	if !bytes.Equal(msg.data, m.Data()) {
		println("Expected", msg.msgid, msg.data, msg.data[0])
		println("Got", m.MsgId(), m.Data(), m.Data()[0])
		println("Error on seq", m.Seq(), "after", self.count, "messages")
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

	r := self.ctx.Channel(fmt.Sprintf("file://%s/file.dat;autoclose=no;io=%s;dump=no;name=reader-%d", self.base, self.io, self.index))
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

	reopen := self.ctx.Channel(fmt.Sprintf("pub+mem://%s/reopen;dump=frame;name=reopen-%d", self.base, self.index))
	self.loop.Add(*reopen)
	defer reopen.Free()
	reopen.CallbackAdd(func(c tll.Channel, m tll.Message) int {
		r.Close()
		if self.count > 0 {
			fmt.Printf("Reader %d: Checked %d messages\n", self.index, self.count)
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
	rcount := flag.Uint("count", 1, "Number of readers")

	flag.Parse()

	lcfg := tll.NewConfig()
	lcfg.Set("type", "spdlog")
	lcfg.Set("levels.tll", "info")
	tll.LoggerConfig(lcfg.ConstConfig)

	s.ctx = tll.NewContext()
	defer s.ctx.Free()

	s.base = "/tmp/"
	s.data = make([]Message, 8192)
	prng := rand.New(rand.NewSource(0xdeadbeef))
	for i := range len(s.data) {
		m := Message{seq: int64(i)}
		m.data = make([]byte, rand.Int31n(1024))
		m.msgid = len(m.data)
		prng.Read(m.data)
		s.data[i] = m
	}

	wcfg := tll.LoadConfigData("url", "file:///tmp/file.dat;dir=w;name=writer;dump=no;version=1")
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
	readers := make([]Reader, int(*rcount))
	for i := range len(readers) {
		readers[i].Settings = s
		readers[i].index = i
		go readers[i].Run()
	}
	for range 100 {
		w.OpenCfg(&ocfg.ConstConfig)

		msg.Data = empty
		cmsg := msg.AsMsg(&pinner)
		reopen.Post(cmsg)
		pinner.Unpin()
		for i := range 1000000 {
			m := s.data[i%8192]
			msg.Seq = int64(i)
			msg.Id = m.msgid
			msg.Data = m.data
			cmsg := msg.AsMsg(&pinner)
			w.Post(cmsg)
			pinner.Unpin()
		}
		w.Close()
	}
	for i := range len(readers) {
		readers[i].loop.SetStop(1)
	}
}
